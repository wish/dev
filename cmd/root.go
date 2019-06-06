package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/wish/dev"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// BuildSha is used by the build to include the git sha in the --version output
	BuildSha = "BuildSha not set (use Makefile to set)"
	// BuildVersion is set by the Makefile with link flags for ouput in --version
	BuildVersion = "Build not set (use Makefile to set)"
	// BuildDate is used by the build to include the build date in the --version output
	BuildDate = "BuildDate not set (use Makefile to set)"
	// config stores all of the configuration data in the dev configuration files
	config = dev.NewConfig()
)

var rootCmd = &cobra.Command{
	Use: "dev",
	Version: "\n" +
		"  Version:\t" + BuildVersion + "\n" +
		"  Built:\t" + BuildDate + "\n" +
		"  Git commit:\t" + BuildSha + "\n" +
		"  OS/Arch:\t" + runtime.GOOS + "/" + runtime.GOARCH,
	Short: "dev is a CLI tool that provides a thin layer of porcelain " +
		"on top of Docker Compose projects.",
}

func configureLogging(logLevel string) {
	if logLevel == "" {
		logLevel = config.Log.Level
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

func runCommand(name string, args []string) {
	log.Debugf("Running: %s %s", name, strings.Join(args, " "))
	command := exec.Command(name, args...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	command.Run()
}

func runDockerCompose(project, cmd string, composePaths []string, args ...string) {
	cmdLine := []string{"-p", project}

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

func runOnContainer(projectName string, project *dev.Project, cmds ...string) {
	cmdLine := []string{"exec"}

	// avoid "input device is not a tty error"
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmdLine = append(cmdLine, "-it")
	}

	cmdLine = append(cmdLine, project.Name)

	for _, cmd := range cmds {
		cmdLine = append(cmdLine, cmd)
	}

	runCommand("docker", cmdLine)
}

func addProjectCommands(projectCmd *cobra.Command, config *dev.Config, project *dev.Project) {
	build := &cobra.Command{
		Use:   "build",
		Short: "Build the " + project.Name + " container (and its dependencies)",
		Run: func(cmd *cobra.Command, args []string) {
			registriesLogin(config)
			runDockerCompose(config.ImagePrefix, "build", project.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(build)

	up := ProjectCmdUpCreate(config, project)
	projectCmd.AddCommand(up)

	ps := &cobra.Command{
		Use:   "ps",
		Short: "List status of " + project.Name + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			runDockerCompose(config.ImagePrefix, "ps", project.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(ps)

	sh := ProjectCmdShCreate(config, project)
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
			// for now we assume the non-shared config is last
			// compose file listed. Needs fixing.
			runDockerCompose(config.ImagePrefix, "down", []string{project.DockerComposeFilenames[i-1]})
		},
	}
	projectCmd.AddCommand(down)
}

func addProjects(cmd *cobra.Command, config *dev.Config) error {
	for _, project := range config.RunnableProjects() {
		log.Debugf("Adding %s to project commands, aliases: %s", project.Name, project.Aliases)
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

	// set default log level, use DEV_LOGS environment variable if
	// specified (info, debug, warn)
	level := viper.GetString("LOGS")
	configureLogging(level)
	initConfig()

	// environment variable takes precendence over config file setting
	if viper.GetString("LOGS") == "" {
		configureLogging(config.Log.Level)
	}

	// removes the annoying: WARNING: Found orphan containers
	// for this project.
	err := os.Setenv("COMPOSE_IGNORE_ORPHANS", "True")
	if err != nil {
		log.Fatalf("Failed to set environment variable: %s", err)
	}

	log.Debugf("Using image prefix '%s'", config.ImagePrefix)

	if err := addProjects(rootCmd, config); err != nil {
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

func getAppConfigPaths(dir string) []string {
	defaultConfigs := make([]string, len(dev.ConfigFileDefaults))
	for i, filename := range dev.ConfigFileDefaults {
		defaultConfigs[i] = path.Join(dir, filename)
	}
	return defaultConfigs
}

func getDefaultAppConfigFilenames() []string {
	return getAppConfigPaths(getDefaultConfigDirectory())
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
	for {
		configFiles := getAppConfigPaths(currentDir)
		for _, configFile := range configFiles {
			if _, err := os.Stat(configFile); err == nil {
				return configFile
			}
		}

		currentDir = path.Clean(path.Join(currentDir, ".."))

		// if we've recursed to the home directory and still no
		// config, let's not go any further..but let's check
		// one more place
		if !strings.Contains(currentDir, home) {
			configFiles := getDefaultAppConfigFilenames()

			for _, configFile := range configFiles {
				if _, err := os.Stat(configFile); err == nil {
					return configFile
				}
			}
			break
		}

	}
	return ""
}

// initConfig locates the configuration file and loads it into the Config
func initConfig() {
	cfgFile := viper.GetString("CONFIG")
	if cfgFile != "" {
		files := strings.Split(cfgFile, ":")
		for _, configFile := range files {
			log.Debugf("Loading env variable specified config file: %s", configFile)
			// viper has merge capabilities, but we'd need to remove the relative
			// paths before the merging
			viper.SetConfigFile(configFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Fatal(err)
			}

			localConfig := dev.NewConfig()
			if err := viper.Unmarshal(localConfig); err != nil {
				log.Fatal(err)
			}
			dev.ExpandConfig(configFile, localConfig)
			if err := dev.MergeConfig(config, localConfig); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		// config file/s not specified in environment variable, see if one
		// can be found
		cfgFile = locateConfigFile()
		if cfgFile != "" {
			log.Debugf("Found config file: %s\n", cfgFile)
			viper.SetConfigFile(cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Fatal(err)
			}
			if err := viper.Unmarshal(&config); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Debugln("No configuration file found")
		}

		dev.ExpandConfig(cfgFile, config)
	}
}
