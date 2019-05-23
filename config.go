package dev

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types"
)

const (
	projectShellDefault           = "/bin/bash"
	registryTimeoutSecondsDefault = 2
	registryContinueOnFail        = false
	// LogLevelDefault is the log level used when one has not been
	// specified in an environment variable or in configuration file.
	LogLevelDefault = "info"
	// NoProjectWarning is the message provided to the user when no project
	// could be found
	NoProjectWarning = `No docker-compose.yml file in this directory.

If you would like to use dev outside of your project directory, create a link
to your project dev.yaml from $HOME or set the DEV_CONFIG environment variable
to point to your project dev.yaml.
`
)

// ConfigFileDefaults are the default filenames for the configuration
// file for this program.
var ConfigFileDefaults = []string{".dev.yml", ".dev.yaml", "dev.yml", "dev.yaml"}

// todo: pull these from docker library if we can
var dockerComposeFilenames = []string{"docker-compose.yml", "docker-compose.yaml"}

// Config is the datastructure into which we unmarshal the dev configuration
// file.
type Config struct {
	Log        LogConfig           `mapstructure:"log"`
	Projects   map[string]*Project `mapstructure:"projects"`
	Registries []*Registry         `mapstructure:"registries"`
	// Filename is the full path of the configuration file containing
	// this configuration. This is used internally and is ignored
	// if specified by the user.
	Filename string
	// Dir is either the location of the config file or the current
	// working directory if there is no config file. This is used
	// intenrally and is ignored if specified by the user.
	Dir string
	// Networks are a list of the networks managed by dev. A network
	// will be created automatically as required by dev if it is listed
	// as a dependency of your project. These are networks that are used
	// as 'external networks' in your docker-compose configuration.
	Networks map[string]*types.NetworkCreate `mapstructure:"networks"`
	// ImagePrefix is the prefix to add to images built with this
	// tool through compose. Compose forces the use of a prefix so we
	// allow the configuration of that prefix here.  Dev must know the
	// prefix in order to perform some image specific operations.  If not
	// set, this defaults to the directory where the this tool's config
	// file is located or the directory or the docker-compose.yml if one is
	// found. Note that compose only adds the prefix to local image
	// builds.
	ImagePrefix string `mapstructure:"image_prefix"`
}

// LogConfig holds the logging related configuration.
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// Project configuration structure. This must be used when using more than one
// docker-compose.yml file for a project.
type Project struct {
	// The paths of the docker compose files for this project. Can be
	// relative or absolute paths.
	DockerComposeFilenames []string `mapstructure:"docker_compose_files"`
	// The path of the root of the project (or its basename). Do we need this?
	Directory string `mapstructure:"directory"`
	// Ignored if set by user.
	Name string `mapstructure:"name"`
	// Alternate names for this project.
	Aliases []string `mapstructure:"aliases"`
	// Whether project should be included for use by this project, default false
	Hidden bool `mapstructure:"hidden"`
	// Shell used to enter the project container with 'sh' command,
	// default is /bin/bash
	Shell string `mapstructure:"shell"`
}

// Registry repesents the configuration required to model a container registry.
// Users can configure their project to be dependent on a registry. When this
// occurs, we will login to the container registry using the configuration
// provided here. This will allow users to host their images in private image
// repos.
type Registry struct {
	// User readable name, not used by the docker client
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
	// TODO: other forms of auth exist and should be supported, but this is
	// what I need..
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`

	// Sometimes these can be firewalled, so a default timeout of 2 seconds
	// is provided, though can be tweaked here
	TimeoutSeconds int64 `mapstructure:"timeout_seconds"`

	// if login or connection fails, should dev continue with command or
	// fail hard.  Default is True
	ContinueOnFailure bool `mapstructure:"continue_on_failure"`
}

// RunnableProjects returns the Project configuration of each Project
// that has a docker-compose.yml file and is not hidden by configuration.
func (c *Config) RunnableProjects() []*Project {
	var projects []*Project

	for _, project := range c.Projects {
		if len(project.DockerComposeFilenames) > 0 && !project.Hidden {
			projects = append(projects, project)
		}
	}
	return projects
}

func pathToDockerComposeFilenames(directory string) []string {
	paths := make([]string, len(dockerComposeFilenames))
	for _, filename := range dockerComposeFilenames {
		paths = append(paths, path.Join(directory, filename))
	}
	return paths
}

func directoryContainsDockerComposeConfig(directory string) (bool, string) {
	for _, configFile := range pathToDockerComposeFilenames(directory) {
		if _, err := os.Stat(configFile); err == nil {
			return true, configFile
		}
	}
	return false, ""
}

func projectNameFromPath(projectPath string) string {
	return path.Base(projectPath)
}

func newProjectConfig(projectPath, composeFilename string) *Project {
	project := &Project{
		Directory:              projectPath,
		Name:                   projectNameFromPath(projectPath),
		DockerComposeFilenames: []string{composeFilename},
	}

	return project
}

func expandRelativeDirectories(config *Config) {
	for _, project := range config.Projects {
		for i, composeFile := range project.DockerComposeFilenames {
			if !strings.HasPrefix(composeFile, "/") {
				fullPath := path.Clean(path.Join(config.Dir, composeFile))
				project.DockerComposeFilenames[i] = fullPath
			}
		}
	}
}

func setDefaults(config *Config) {
	// Need to be smarter here.. Users unable to specify 0 here, which is
	// a reasonable default for many values.
	for _, registry := range config.Registries {
		if registry.TimeoutSeconds == 0 {
			registry.TimeoutSeconds = registryTimeoutSecondsDefault
		}
	}

	for _, project := range config.Projects {
		if project.Shell == "" {
			project.Shell = projectShellDefault
		}
	}

	if config.Log.Level == "" {
		config.Log.Level = LogLevelDefault
	}

	if config.ImagePrefix == "" {
		config.ImagePrefix = filepath.Base(config.Dir)
	}
}

// ExpandConfig makes modifications to the configuration structure
// provided by the user before it is used by dev.
func ExpandConfig(filename string, config *Config) {
	if config.Projects == nil {
		config.Projects = make(map[string]*Project)
	}

	// Ensure that relative paths used in the configuration file are
	// relative to the actual project, not to the location of a link.
	if filename != "" {
		fi, err := os.Lstat(filename)
		if err != nil {
			log.Fatalf("Error fetching file info for %s: %s", filename, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			if filename, err = os.Readlink(filename); err != nil {
				log.Fatalf("ReadLink error for config file: %s", filename)
			}
		}
	}

	config.Filename = filename
	if config.Filename == "" {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Error getting the current directory: %s", err)
		}
		config.Dir = dir
	} else {
		config.Dir = filepath.Dir(config.Filename)
	}

	expandRelativeDirectories(config)

	// for _, project := range config.Projects {
	// 	for _, composeFilename := range project.DockerComposeFilenames {
	// 		log.Debugf("hmm2: %s", composeFilename)
	// 	}
	// }
	setDefaults(config)

	for name, project := range config.Projects {
		project.Name = name
	}

	// If there's a docker-compose file in the current directory
	// that's not specified in the config file create a default project
	// with the name of the directory.
	if hasConfig, filename := directoryContainsDockerComposeConfig(config.Dir); hasConfig {
		found := false
		for _, project := range config.Projects {
			if project.Directory == config.Dir {
				found = true
			}
		}
		if found == false {
			log.Debugf("Creating default project config for project in %s", config.Dir)
			project := newProjectConfig(config.Dir, filename)
			config.Projects[project.Name] = project
		}
	}

	if len(config.Projects) == 0 {
		fmt.Println(NoProjectWarning)
	}
}
