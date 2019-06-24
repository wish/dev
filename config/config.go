package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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

// Dev is the datastructure into which we unmarshal the dev configuration
// file.
type Dev struct {
	Log        LogConfig            `mapstructure:"log"`
	Projects   map[string]*Project  `mapstructure:"projects"`
	Registries map[string]*Registry `mapstructure:"registries"`
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
	// Directory is the full-path to the location of the dev configuration
	// file that contains this project configuration.
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
	// Projects, registries, networks on which this project depends.
	Dependencies []string `mapstructure:"depends_on"`
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

// NewConfig structs the default configuration structure for dev driven
// projects.
func NewConfig() *Dev {
	config := &Dev{
		Projects:   make(map[string]*Project),
		Networks:   make(map[string]*types.NetworkCreate),
		Registries: make(map[string]*Registry),
		Log: LogConfig{
			Level: LogLevelDefault,
		},
	}
	return config
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
		Directory: projectPath,
		Name:      projectNameFromPath(projectPath),
		DockerComposeFilenames: []string{composeFilename},
	}

	return project
}

func expandRelativeDirectories(config *Dev) {
	for _, project := range config.Projects {
		for i, composeFile := range project.DockerComposeFilenames {
			if !strings.HasPrefix(composeFile, "/") {
				fullPath := path.Clean(path.Join(config.Dir, composeFile))
				project.DockerComposeFilenames[i] = fullPath
			}
		}
	}
}

func setDefaults(config *Dev) {
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

	if config.ImagePrefix == "" {
		config.ImagePrefix = filepath.Base(config.Dir)
	}

	for name, registry := range config.Registries {
		registry.Name = name
	}

	for name, project := range config.Projects {
		// if user did not specify a custom project container name,
		// the container name is assumed to be the same name as the
		// project itself.
		if project.Name == "" {
			project.Name = name
		}
		// Have to remember where the project was defined so we can
		// figure out relative directories when launching commands. See
		// Project.Shell. This is also passed in when parsing docker-compose
		// files where it used to load env files.
		project.Directory = filepath.Dir(config.Filename)
	}
}

// Expand makes modifications to the configuration structure
// provided by the user before it is used by dev.
func Expand(filename string, config *Dev) {
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

	for name, registry := range config.Registries {
		registry.Name = name
	}
	for name, project := range config.Projects {
		// if user did not specify a custom project container name,
		// the container name is assumed to be the same name as the
		// project itself. Should probably split this out into another
		// variable.
		if project.Name == "" {
			project.Name = name
		}
	}

	// The code below does not work because project.Directory is not set!
	// Temporarily working around by only adding projects when a dev config
	// file is not found.
	if config.Filename == "" {
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
	}

	if len(config.Projects) == 0 {
		fmt.Println(NoProjectWarning)
	}
}

func isDefaultConfig(config *Dev) bool {
	return (len(config.Projects) == 0 && len(config.Networks) == 0 &&
		len(config.Registries) == 0 && config.ImagePrefix == "")
}

// Merge adds the configuration from source to the configuration from
// target. An error is returned if there is an object with the same name
// in target and source or if the configs cannot be merged for whatever reason.
func Merge(target *Dev, source *Dev) error {
	if isDefaultConfig(target) {
		// project wide settings are set by the first config listed
		target.ImagePrefix = source.ImagePrefix
		target.Log.Level = source.Log.Level
		target.Dir = source.Dir
		target.Filename = source.Filename

	} else if source.ImagePrefix != target.ImagePrefix {
		// Not sure I like forcing this.. but if users switch back and forth
		// between a project that uses multiple configurations and then start
		// using one of those with only one configuration, some dev commands
		// will not function b/c docker will change the name of the container
		// b/c it's using a different image name, usually appending a _#.
		return errors.Errorf("mismatched image prefix '%s' != '%s'", target.ImagePrefix, source.ImagePrefix)
	}

	for _, project := range source.Projects {
		if _, exists := target.Projects[project.Name]; exists {
			return errors.Errorf("duplicate project with name %s found in %s", project.Name, source.Filename)
		}
	}
	for _, project := range source.Projects {
		target.Projects[project.Name] = project
	}

	for name := range source.Networks {
		if _, ok := target.Networks[name]; ok {
			return errors.Errorf("duplicate network with name %s found in %s", name, source.Filename)
		}
	}
	for name, network := range source.Networks {
		target.Networks[name] = network
	}

	for name := range source.Registries {
		if _, ok := target.Registries[name]; ok {
			return errors.Errorf("duplicate registry with name %s found in %s", name, source.Filename)
		}
	}
	for name, registry := range source.Registries {
		target.Registries[name] = registry
	}

	return nil
}
