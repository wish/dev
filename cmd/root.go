package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/wish/dev"
	"github.com/wish/dev/registry"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// BuildSha is used by the build to include the git sha in the --version output
	BuildSha = "BuildSha not set (use Makefile to set)"
	// BuildDate is used by the build to include the build date in the --version output
	BuildDate = "BuildDate not set (use Makefile to set)"
	config    dev.Config
)

const (
	// configFilename is the filename of the default configuration file
	configFilename = ".dev.toml"
)

var rootCmd = &cobra.Command{
	Use: "dev",
	Version: "\n" +
		"  Built:\t" + BuildDate + "\n" +
		"  Git commit:\t" + BuildSha + "\n" +
		"  OS/Arch:\t" + runtime.GOOS + "/" + runtime.GOARCH,
	Short: "dev is a CLI tool that provides a thin layer of porcelain " +
		"on top of Docker Compose projects.",
}

func configureLogging(logLevel string) {
	if logLevel == "" {
		logLevel = dev.LogLevelDefault
	}
	logRusLevel, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logRusLevel)
	log.SetOutput(os.Stderr)
	// print after setting so this shows up/doesn't show up as appropriate
	log.Debugf("Set logging to %s", logLevel)
}

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to happen once
// to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func quote(str string) string {
	return fmt.Sprintf("%s", str)
}

func runCommand(name string, args []string) {
	log.Debugf("Running: %s %s", name, strings.Join(args, " "))
	command := exec.Command(name, args...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	command.Run()
}

func runDockerCompose(cmd string, composePaths []string, args ...string) {
	var cmdLine []string

	for _, path := range composePaths {
		cmdLine = append(cmdLine, "-f", path)
	}
	// one of build, exec, etc.
	cmdLine = append(cmdLine, cmd)

	// append any additional arguments or flags, i.e., -d
	for _, arg := range args {
		cmdLine = append(cmdLine, arg)
	}

	runCommand("docker-compose", cmdLine)
}

func runOnContainer(project *dev.Project, cmds ...string) {
	var cmdLine []string

	for _, path := range project.DockerComposeFilenames {
		cmdLine = append(cmdLine, "-f", path)
	}
	cmdLine = append(cmdLine, "exec", project.Name)

	// append any additional arguments or flags, i.e., -d
	for _, cmd := range cmds {
		cmdLine = append(cmdLine, cmd)
	}

	runCommand("docker-compose", cmdLine)
}

func addProjectCommands(projectCmd *cobra.Command, config *dev.Config, project *dev.Project) {
	build := &cobra.Command{
		Use:   "build",
		Short: "Build the " + project.Name + " container (and its dependencies)",
		Run: func(cmd *cobra.Command, args []string) {
			runDockerCompose("build", project.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(build)

	up := &cobra.Command{
		Use:   "up",
		Short: "Create and start the " + project.Name + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			for _, r := range config.Registries {
				err := registry.Login(r)
				if err != nil {
					msg := fmt.Sprintf("Failed to login to %s registry: %s", r.Name, err)
					if r.ContinueOnFailure {
						log.Warn(msg)
					} else {
						log.Fatal(msg)
					}

				}
			}
			runDockerCompose("up", project.DockerComposeFilenames, "-d")
			runDockerCompose("logs", project.DockerComposeFilenames, "-f", project.Name)
		},
	}
	projectCmd.AddCommand(up)

	ps := &cobra.Command{
		Use:   "ps",
		Short: "List status of " + project.Name + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			runDockerCompose("ps", project.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(ps)

	// needs work here... to pass args, gotta quote everything... -- doesn't work, etc.
	// if working directory is within the project, then the context of the command
	// should match...should
	sh := &cobra.Command{
		Use:   "sh",
		Short: "Get a shell on the " + project.Name + " container",
		Args:  cobra.ArbitraryArgs,
		// Need to handle the flags manually. We do this so that we can
		// send in flags to the container without quoting the entire
		// string-- in the name of usability.
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current directory, attempt to find its location
			// on the container and cd to it. This allows developers to
			// use relative directories like they would in a non-containerized
			// development environment.
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("Failed to get current directory: %s", err)
			}
			configDir := filepath.Dir(config.Filename)

			relativePath := ""
			if strings.HasPrefix(cwd, configDir) {
				start := strings.Index(cwd, configDir) + len(configDir) + 1
				if start < len(cwd) {
					relativePath = cwd[start:]
				} else {
					relativePath = "."
				}
			}

			if len(args) > 0 {
				// assume a command starting with a dash is
				// a cry for help. Make this smarter..
				if strings.HasPrefix(args[0], "-") {
					cmd.Help()
					return
				}
			} else {
				// no subcommands, so just provide a shell
				args = append(args, project.Shell)
			}

			cmdLine := []string{project.Shell, "-c",
				fmt.Sprintf("cd %s ; %s", relativePath, strings.Join(args, " "))}

			runOnContainer(project, cmdLine...)
		},
	}
	projectCmd.AddCommand(sh)

	down := &cobra.Command{
		Use:   "down",
		Short: "Stop and destroy the " + project.Name + " project container",
		Long: `This stops and destroys the container of the same name as the directory in which
its docker-compose.yml file is placed. It does not stop or destroy any containers that
may have been brought up to support this project, which is the case for projects that
use more one docker-compose.yml file.`,
		Run: func(cmd *cobra.Command, args []string) {
			i := len(project.DockerComposeFilenames)
			// for now we assume that the project configuration is in the last compose
			// file listed, which it should be.. would be better to parse the configs
			// and verify assumptions.
			runDockerCompose("down", []string{project.DockerComposeFilenames[i-1]})
		},
	}
	projectCmd.AddCommand(down)
}

func addProjects(cmd *cobra.Command, config *dev.Config) error {
	for _, project := range config.RunnableProjects() {
		log.Debugf("Adding %s to project commands", project.Name)
		cmd := &cobra.Command{
			Use:     project.Name,
			Short:   "Run dev commands on the " + project.Name + " project",
			Aliases: project.Aliases,
		}
		rootCmd.AddCommand(cmd)
		addProjectCommands(cmd, config, project)
	}
	return nil
}

func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("DEV")

	if err := viper.BindEnv("CONFIG"); err != nil {
		log.Fatalf("error binding to DEV_CONFIG environment variable: %s", err)
	}
	if err := viper.BindEnv("LOGS"); err != nil {
		log.Fatalf("error binding to DEV_LOGS environment variable: %s", err)
	}

	// XXX: no global command line flags (persistentFlags) b/c they
	// DisableFlagParsing is set for the 'sh' command so users do not have to
	// surround command line with quotes or preceed with --.

	// set default log level, use DEV_LOGS environment variable if specified (info, debug, warn)
	level := viper.GetString("LOGS")
	configureLogging(level)
	initConfig()
	// reconfigure logging with config from config file
	configureLogging(config.Log.Level)

	// adjust log level config file specification

	// following removes the following: WARNING: Found orphan containers for this project.
	err := os.Setenv("COMPOSE_IGNORE_ORPHANS", "True")
	if err != nil {
		log.Fatalf("Failed to set environment variable: %s", err)
	}

	if err := addProjects(rootCmd, &config); err != nil {
		log.Fatalf("Error adding projects: %s", err)
	}
}

// getDefaultConfigDirectory returns the configuration directory of this tool.
// The directory returned may not exist.
func getDefaultConfigDirectory() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, _ := homedir.Dir()
		configHome = path.Join(homeDir, ".config")
	}
	return path.Join(configHome, "dev")
}

// getdefaultAppConfigFilename returns the full path and filename of the
// default configuration file for this tool. This file is only consulted
// when there is no project-level configuration file.
func getDefaultAppConfigFilename() string {
	return path.Join(getDefaultConfigDirectory(), configFilename)
}

// locateConfigFile attempts to locate the path at which the configuration file
// for this program exists. The full path is returned if a configuration file
// is found, otherwise an empty string is returned.
func locateConfigFile() string {
	// search upward from the current directory until we get a
	// configuration file; stop when we've passed the users home directory.
	// Should probably go all the way to root if current directory is
	// outside of the users home directory.
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	currentDir := dir
	currentConfigFile := ""
	devConfig := ""
	for {
		currentConfigFile = path.Join(currentDir, configFilename)
		if _, err := os.Stat(currentConfigFile); err == nil {
			devConfig = currentConfigFile
			break
		} else {
			currentDir = path.Clean(path.Join(currentDir, ".."))
			// we've recursed to the home directory and still no
			// config, let's not go any further
			if !strings.Contains(currentDir, home) {
				defaultConfig := getDefaultAppConfigFilename()
				if _, err := os.Stat(getDefaultAppConfigFilename()); err == nil {
					devConfig = defaultConfig
				}
				break
			}
		}
	}
	return devConfig
}

// initConfig locates the configuration file and loads it into the Config
func initConfig() {
	cfgFile := viper.GetString("FILE")
	if cfgFile != "" {
		log.Debugf("Using command line specified config file: %s", cfgFile)
		// specified on the command line
		viper.SetConfigFile(cfgFile)
	} else {
		// try to locate a configuration file. Enables you to use the
		// tool for more than one project a time or all together..
		cfgFile = locateConfigFile()
		if cfgFile != "" {
			log.Debugf("Found config file: %s\n", cfgFile)
			viper.SetConfigFile(cfgFile)
		} else {
			log.Debugln("No configuration file found")
		}
	}

	if cfgFile != "" {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal(err)
	}

	dev.ExpandConfig(cfgFile, &config)
}
