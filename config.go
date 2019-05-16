package dev

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	projectSearchDepthDefault     = 2
	projectShellDefault           = "/bin/bash"
	registryTimeoutSecondsDefault = 2
	registryContinueOnFail        = false
	// LogLevelDefault is the log level used when one has not been
	// specified in an environment variable or in a configuration file.
	LogLevelDefault = "info"
	// ConfigFileDefault is the default filename for the configuration file
	// for this program.
	ConfigFileDefault = ".dev.yml"
)

var directoriesDefault = []string{"."}

// Config is the datastructure into which we unmarshal the dev configuration
// file.
type Config struct {
	Log LogConfig `mapstructure:"log"`
	// List of directories to search for docker-compose.yml files
	ProjectDirectories []string    `mapstructure:"directories"`
	Projects           []*Project  `mapstructure:"projects"`
	Registries         []*Registry `mapstructure:"registries"`
	// Filename is the full path of the configuration file
	Filename string
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
	Name      string `mapstructure:"name"`
	// Alternate names for this project
	Aliases []string `mapstructure:"aliases"`
	// Whether project should be included for use by this project, default false
	Hidden bool `mapstructure:"hidden"`
	// The number of sub-directories undeath a Project directory that is
	// searched for DockerCompose files.
	SearchDepth int `mapstructure:"depth"`
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

func locateDockerComposeFiles(startDirectory string, depth int) []string {
	var configs []string

	startDepth := strings.Count(startDirectory, "/")
	//composeFilename := path.Join(startDirectory, "docker_compose.yml")
	filepath.Walk(startDirectory, func(pathname string, info os.FileInfo, _ error) error {
		endDepth := strings.Count(pathname, "/")

		if endDepth-startDepth > depth {
			//log.Debugf("Max depth reached, skipping: %s", pathname)
			return filepath.SkipDir
		}

		if info.IsDir() {
			if strings.HasPrefix(path.Base(pathname), ".") {
				//log.Debugf("Skipping hidden directory: %s", pathname)
				return filepath.SkipDir
			}
			if directoryContainsDockerComposeConfig(pathname) {
				//log.Debugf("Found %s", dockerComposeFullPath(pathname))
				configs = append(configs, dockerComposeFullPath(pathname))
			}
		}
		return nil
	})

	return configs
}

func projectNameFromPath(projectPath string) string {
	return path.Base(projectPath)
}

// getOrCreateProjectConfig returns the existing Project struct for the
// specified project path (i.e., the full path to the project). Will create a
// Project configuration if there is not an existing user-provided one.
func getOrCreateProjectConfig(config *Config, projectPath string) *Project {
	for _, project := range config.Projects {
		//log.Infof("Found project with name: %s", project.Name)
		found := false
		if project.Directory == projectPath {
			log.Infof("Found project by path: %s, setting name to %s",
				project.Directory, projectNameFromPath(projectPath))
			project.Name = projectNameFromPath(projectPath)
			found = true
		}
		if project.Name == projectNameFromPath(projectPath) {
			log.Debugf("Found project by name: %s; setting project directory to %s",
				project.Name, projectPath)
			project.Directory = projectPath
			found = true
		}
		if found == true {
			composePath := dockerComposeFullPath(projectPath)
			if !SliceContainsString(project.DockerComposeFilenames, composePath) {
				project.DockerComposeFilenames = append(project.DockerComposeFilenames, composePath)
			}
			return project
		}
	}
	log.Debugf("Did not find existing project configuration for %s, creating", projectNameFromPath(projectPath))

	project := &Project{
		Directory:   projectPath,
		Name:        projectNameFromPath(projectPath),
		SearchDepth: projectSearchDepthDefault,
	}
	if directoryContainsDockerComposeConfig(projectPath) {
		composePath := dockerComposeFullPath(projectPath)
		project.DockerComposeFilenames = append(project.DockerComposeFilenames, composePath)
	}

	config.Projects = append(config.Projects, project)
	return project
}

func expandProject(config *Config, project *Project) {
	if project.Directory != "" && !project.Hidden {
		composeFiles := locateDockerComposeFiles(project.Directory, project.SearchDepth)
		log.Debugf("(%s) Found docker_compose.yml files: %s", project.Directory, strings.Join(composeFiles, ", "))
		for _, composePath := range composeFiles {
			getOrCreateProjectConfig(config, path.Dir(composePath))
		}
	}
}

func expand(config *Config) {
	// See if any evironment variables are used in the Project
	// Directories and expand as necessary.
	for i, dir := range config.ProjectDirectories {
		config.ProjectDirectories[i] = os.ExpandEnv(dir)
	}

	// Expand environment variables used in project directories..
	for _, project := range config.Projects {
		project.Directory = os.ExpandEnv(project.Directory)
	}

	// Expand environment vars used in docker_compose_file locations
	for _, project := range config.Projects {
		for i, composeFile := range project.DockerComposeFilenames {
			project.DockerComposeFilenames[i] = os.ExpandEnv(composeFile)
		}
	}
}

func expandRelativeDirectories(config *Config) {
	if len(config.ProjectDirectories) == 0 {
		config.ProjectDirectories = directoriesDefault
	}

	for i, dir := range config.ProjectDirectories {
		if !strings.HasPrefix(dir, "/") {
			var configDir string
			if config.Filename == "" {
				configDir, _ = os.Getwd()
			} else {
				configDir = filepath.Dir(config.Filename)
			}
			config.ProjectDirectories[i] = path.Clean(path.Join(configDir, dir))
		}
	}

	for _, project := range config.Projects {
		for i, composeFile := range project.DockerComposeFilenames {
			if !strings.HasPrefix(composeFile, "/") {
				configDir := filepath.Dir(config.Filename)
				project.DockerComposeFilenames[i] = path.Clean(path.Join(configDir, composeFile))
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
	// Ensure that relative paths used in the configuration file are relative
	// the actual project, not to the location of a link.
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

	expandRelativeDirectories(config)
	expand(config)
	setDefaults(config)

	// Find individual projects by locating docker-compose.yml files in the
	// specified project directories.  Create/synchronize a Project
	// configuration for each found Project.
	for _, projectDir := range config.ProjectDirectories {
		// see if there is a project configuration
		getOrCreateProjectConfig(config, projectDir)
	}

	for _, project := range config.Projects {
		expandProject(config, project)
	}
}
