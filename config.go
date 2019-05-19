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
	// specified in an environment variable or in a configuration file.
	LogLevelDefault = "info"
	// ConfigFileDefault is the default filename for the configuration file
	// for this program.
	ConfigFileDefault = ".dev.yml"
	// NoProjectWarning is the message provided to the user when no project
	// could be found
	NoProjectWarning = `Unable to locate any docker-compose.yml files in this directory.

If you would like to use dev outside of your project directory, create a link
to your project .dev.yml from $HOME.
`
)

// Config is the datastructure into which we unmarshal the dev configuration
// file.
type Config struct {
	Log        LogConfig           `mapstructure:"log"`
	Projects   map[string]*Project `mapstructure:"projects"`
	Registries []*Registry         `mapstructure:"registries"`
	// Filename is the full path of the configuration file
	Filename string
	// Dir is either the location of the config file or the current
	// working directory if there is no config file.
	Dir string
	// Networks are a list of the networks managed by dev. A network
	// will be created automatically as required by dev if it is listed
	// as a dependency of your project. These are networks that are used
	// as 'external networks' in your docker-compose configuration.
	Networks map[string]*types.NetworkCreate `mapstructure:"networks"`
}

// LogConfig holds the logging related configuration.
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// Project configuration structure. This must be used when using more than one
// docker-compose.yml file for a project.
type Project struct {
	// The absolute paths of the docker compose files for this project. If
	// not specified, project directories will be searched for one. If the
	// project needs multiple, they must be specified.
	DockerComposeFilenames []string `mapstructure:"docker_compose_files"`
	// The absolute path of the root of the project (or its basename). This
	// need not be the same as a directory in ProjectDirectories, but must
	// be a child directory of one of those paths. If not specified, this
	// directory is assumed to be at the same location as the
	// DockerCompose.yml file.
	Directory string `mapstructure:"directory"`
	// TODO: unused remove. Currently set to the key name.
	Name string `mapstructure:"name"`
	// Alternate names for this project
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

func dockerComposeFullPath(directory string) string {
	return path.Join(directory, "docker-compose.yml")
}

func directoryContainsDockerComposeConfig(directory string) bool {
	composeFilename := dockerComposeFullPath(directory)
	if _, err := os.Stat(composeFilename); err == nil {
		return true
	}
	return false
}

func projectNameFromPath(projectPath string) string {
	return path.Base(projectPath)
}

func newProjectConfig(projectPath string) *Project {
	project := &Project{
		Directory: projectPath,
		Name:      projectNameFromPath(projectPath),
		DockerComposeFilenames: []string{dockerComposeFullPath(projectPath)},
	}

	return project
}

func expandRelativeDirectories(config *Config) {
	for _, project := range config.Projects {
		for i, composeFile := range project.DockerComposeFilenames {
			if !strings.HasPrefix(composeFile, "/") {
				project.DockerComposeFilenames[i] = path.Clean(path.Join(config.Dir, composeFile))
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
}

// ExpandConfig makes modifications to the configuration structure
// provided by the user before it is used by dev-cli.
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
	setDefaults(config)

	for name, project := range config.Projects {
		project.Name = name
	}

	// If there's a docker compose file in the current directory
	// that's not specified in the config file create a default project
	// with the name of the directory.
	if directoryContainsDockerComposeConfig(config.Dir) {
		found := false
		for _, project := range config.Projects {
			if project.Directory == config.Dir {
				found = true
			}
		}
		if found == false {
			log.Debugf("Creating default project config for project in %s", config.Dir)
			project := newProjectConfig(config.Dir)
			config.Projects[project.Name] = project
		}
	}

	if len(config.Projects) == 0 {
		fmt.Println(NoProjectWarning)
	}
}
