package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/wish/dev"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile, logLevel  string
	BuildSha           string = "BuildSha not set (use Makefile to set)"
	BuildDate          string = "BuildDate not set (use Makefile to set)"
	Config             dev.Config
	projectDirectories string
)

const (
	// name of the default configuration file
	ConfigFilename = ".dev.toml"
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

func configureLogging() {
	level := viper.GetString("log.level")
	//fmt.Printf("configureLogging: log.level: %s loglevel: %s\n", level, logLevel)
	logRusLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logRusLevel)
	log.SetOutput(os.Stderr)
	// print after setting so this shows up/doesn't show up as appropriate
	log.Debugf("Set logging to %s", level)
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

	log.Debugf("Running command: %s", strings.Join(cmdLine, " "))
	command := exec.Command("docker-compose", cmdLine...)

	// check this....
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	command.Run()
}

func addProjectCommands(projectCmd *cobra.Command, config *dev.Project) {
	build := &cobra.Command{
		Use:   "build",
		Short: "Build the " + config.GetProjectIdentifier() + " container (and its dependencies)",
		Run: func(cmd *cobra.Command, args []string) {
			runDockerCompose("build", config.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(build)

	up := &cobra.Command{
		Use:   "up",
		Short: "Start the " + config.GetProjectIdentifier() + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			runDockerCompose("up", config.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(up)

	ps := &cobra.Command{
		Use:   "ps",
		Short: "List status of " + config.GetProjectIdentifier() + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			runDockerCompose("ps", config.DockerComposeFilenames)
		},
	}
	projectCmd.AddCommand(ps)
}

func addProjects(cmd *cobra.Command, config *dev.Config) error {
	for _, project := range config.RunnableProjects() {
		projectName := project.GetProjectIdentifier()
		log.Debugf("Adding %s to project commands", projectName)
		cmd := &cobra.Command{
			Use:   projectName,
			Short: "Run dev commands on the " + projectName + " project",
		}
		rootCmd.AddCommand(cmd)
		addProjectCommands(cmd, project)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "debug", "log level (warn, info, debug)")
	rootCmd.PersistentFlags().StringVar(&projectDirectories, "directories",
		".", "Directories to search for docker-compose.yml files")

	if err := viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log")); err != nil {
		fmt.Printf("Error binding to log.level %s", err)
		os.Exit(1)
	}

	// Set logging based on command line only (this is not working)
	configureLogging()

	if err := viper.BindPFlag("directories", rootCmd.PersistentFlags().Lookup("directories")); err != nil {
		fmt.Printf("Error binding to directories %s", err)
		os.Exit(1)
	}

	initConfig()

	// Take configuration file into account for logging preference
	configureLogging()

	if err := addProjects(rootCmd, &Config); err != nil {
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
	return path.Join(getDefaultConfigDirectory(), ConfigFilename)
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
		currentConfigFile = path.Join(currentDir, ConfigFilename)
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
	viper.AutomaticEnv()
	viper.SetEnvPrefix("DEV")

	if cfgFile != "" {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}

	if err := viper.Unmarshal(&Config); err != nil {
		log.Fatal(err)
	}

	dev.ExpandConfig(&Config)
}
