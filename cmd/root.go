package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/wish/dev"
	"github.com/wish/dev/config"
)

var (
	// BuildSha is used by the build to include the git sha in the --version output
	BuildSha = "BuildSha not set (use Makefile to set)"
	// BuildVersion is set by the Makefile with link flags for ouput in --version
	BuildVersion = "Build not set (use Makefile to set)"
	// BuildDate is used by the build to include the build date in the --version output
	BuildDate = "BuildDate not set (use Makefile to set)"
	// appConfig stores all of the configuration data in the dev configuration files
	appConfig = config.NewConfig()
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
		logLevel = appConfig.Log.Level
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

func addProjectCommands(projectCmd *cobra.Command, devConfig *config.Dev, project *dev.Project) {
	build := &cobra.Command{
		Use:   dev.BUILD,
		Short: "Build the " + project.Name + " container (and its dependencies)",
		PreRun: func(cmd *cobra.Command, args []string) {
			if err := dev.InitDeps(appConfig, dev.BUILD, project); err != nil {
				log.Fatalf("dependency initialization error: %s", err)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			dev.RunComposeBuild(
				devConfig.ImagePrefix,
				project.Config.DockerComposeFilenames,
			)
		},
	}
	projectCmd.AddCommand(build)

	up := &cobra.Command{
		Use:   dev.UP,
		Short: "Create and start the " + project.Name + " containers",
		PreRun: func(cmd *cobra.Command, args []string) {
			if err := dev.InitDeps(appConfig, dev.UP, project); err != nil {
				log.Fatalf("dependency initialization error: %s", err)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			project.UpFollowProjectLogs(appConfig)
		},
	}
	projectCmd.AddCommand(up)

	ps := &cobra.Command{
		Use:   dev.PS,
		Short: "List status of " + project.Name + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			dev.RunComposePs(
				devConfig.ImagePrefix,
				project.Config.DockerComposeFilenames,
			)
		},
	}
	projectCmd.AddCommand(ps)

	sh := &cobra.Command{
		Use:   dev.SH,
		Short: "Get a shell on the " + project.Name + " container",
		Args:  cobra.ArbitraryArgs,
		// Need to handle the flags manually. We do this so that we can
		// send in flags to the container without quoting the entire
		// string-- in the name of usability.
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			// move this to args()
			if len(args) > 0 && strings.HasPrefix(args[0], "-") {
				cmd.Help()
				return
			}
			project.Shell(appConfig, args)
		},
	}
	projectCmd.AddCommand(sh)

	down := &cobra.Command{
		Use:   dev.DOWN,
		Short: "Stop and destroy the " + project.Name + " project container",
		Long: `This stops and destroys the container of the same name as the directory in which
its docker-compose.yml file is placed. It does not stop or destroy any containers that
may have been brought up to support this project, which is the case for projects that
use more one docker-compose.yml file.`,
		Run: func(cmd *cobra.Command, args []string) {
			i := len(project.Config.DockerComposeFilenames)
			// for now we assume the non-shared config is last
			// compose file listed. Needs fixing.
			dev.RunComposeDown(
				devConfig.ImagePrefix,
				[]string{project.Config.DockerComposeFilenames[i-1]},
			)
		},
	}
	projectCmd.AddCommand(down)
}

func addProjects(cmd *cobra.Command, config *config.Dev) error {
	for _, projectConfig := range config.Projects {
		log.Debugf("Adding %s to project commands, aliases: %s", projectConfig.Name, projectConfig.Aliases)
		cmd := &cobra.Command{
			Use:     projectConfig.Name,
			Short:   "Run dev commands on the " + projectConfig.Name + " project",
			Aliases: projectConfig.Aliases,
			Hidden:  projectConfig.Hidden,
		}
		rootCmd.AddCommand(cmd)

		project := dev.NewProject(projectConfig)
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
	initConfig(appConfig)

	// environment variable takes precendence over config file setting
	if viper.GetString("LOGS") == "" {
		configureLogging(appConfig.Log.Level)
	}

	// removes the annoying: WARNING: Found orphan containers
	// for this project.
	err := os.Setenv("COMPOSE_IGNORE_ORPHANS", "True")
	if err != nil {
		log.Fatalf("Failed to set environment variable: %s", err)
	}

	log.Debugf("Using image prefix '%s'", appConfig.ImagePrefix)

	if err := addProjects(rootCmd, appConfig); err != nil {
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
	defaultConfigs := make([]string, len(config.ConfigFileDefaults))
	for i, filename := range config.ConfigFileDefaults {
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

func followLink(filename string) string {
	fi, err := os.Lstat(filename)
	if err != nil {
		log.Fatalf("Error fetching file info for %s: %s", filename, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		if filename, err = os.Readlink(filename); err != nil {
			log.Fatalf("ReadLink error for config file: %s", filename)
		}
	}
	return filename
}

// initConfig locates the configuration file and loads it into the Config
func initConfig(devConfig *config.Dev) {
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
			localConfig := config.NewConfig()
			if err := viper.Unmarshal(localConfig); err != nil {
				log.Fatal(err)
			}
			// Ensure that relative paths used in the configuration
			// file are relative to the actual project, not to the
			// location of a link by following any link provided as
			// a configuration file.
			configFile = followLink(configFile)
			config.Expand(configFile, localConfig)
			if err := config.Merge(devConfig, localConfig); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		// config file/s not specified in environment variable. see if one
		// can be found
		cfgFile = locateConfigFile()
		if cfgFile != "" {
			log.Debugf("Found config file: %s\n", cfgFile)
			viper.SetConfigFile(cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Fatal(err)
			}
			if err := viper.Unmarshal(devConfig); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Debugln("No configuration file found")
		}

		cfgFile = followLink(cfgFile)
		config.Expand(cfgFile, devConfig)
	}
}
