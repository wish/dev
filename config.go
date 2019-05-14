package dev

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const DefaultSearchDepth = 2

type Config struct {
	Log                LogConfig  `mapstructure:"log"`
	ProjectDirectories []string   `mapstructure:"directories"`
	Projects           []*Project `mapstructure:"projects"`

	// Full path and filename of docker-compose.yml files in the source
	// trees specified in the ProjectDirectories list.
	ComposeFiles []string
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

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
	// A shorter name for this project for the generated sub-commands.
	Alias string `mapstructure:"alias"`
	// Whether project should be included for use by this project, default false
	Hidden bool `mapstructure:"hidden"`
	// The number of sub-directories undeath a Project directory that is
	// searched for DockerCompose files.
	SearchDepth int `mapstructure: "depth"`
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

func (p *Project) GetProjectIdentifier() string {
	if p.Alias != "" {
		return p.Alias
	}
	return p.Name
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
func getOrCreateProjectConfig(devConfig *Config, projectPath string) *Project {
	for _, project := range devConfig.Projects {
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
		SearchDepth: DefaultSearchDepth,
	}
	if directoryContainsDockerComposeConfig(projectPath) {
		composePath := dockerComposeFullPath(projectPath)
		project.DockerComposeFilenames = append(project.DockerComposeFilenames, composePath)
	}

	devConfig.Projects = append(devConfig.Projects, project)
	return project
}

func expandProject(devConfig *Config, project *Project) {
	if project.Directory != "" && !project.Hidden {
		composeFiles := locateDockerComposeFiles(project.Directory, project.SearchDepth)
		log.Debugf("(%s) Found docker_compose.yml files: %s", project.Directory, strings.Join(composeFiles, ", "))
		for _, composePath := range composeFiles {
			getOrCreateProjectConfig(devConfig, path.Dir(composePath))
		}
	}
}

// ExpandConfig makes modifications to the configuration structure
// provided by the user before it is used by dev-cli.
func ExpandConfig(devConfig *Config) {
	// See if any evironment variables are used in the Project
	// Directories and expand as necessary.
	for i, dir := range devConfig.ProjectDirectories {
		if strings.HasPrefix(dir, "$") {
			devConfig.ProjectDirectories[i] = os.Getenv(dir[1:])
		}
	}

	// Expand environment variables used in project directories..
	for _, project := range devConfig.Projects {
		if strings.HasPrefix(project.Directory, "$") {
			// fragile..needs improvment
			end := strings.Index(project.Directory, "/")
			if end == -1 {
				project.Directory = os.Getenv(project.Directory[1:])
			} else {
				dir := os.Getenv(project.Directory[1:end])
				project.Directory = path.Join(dir, project.Directory[end:])
			}
		}
	}

	// Expand environment vars used in docker_compose_file locations
	for _, project := range devConfig.Projects {
		for i, composeFile := range project.DockerComposeFilenames {
			if strings.HasPrefix(composeFile, "$") {
				end := strings.Index(composeFile, "/")
				if end == -1 {
					project.DockerComposeFilenames[i] = os.Getenv(composeFile[1:])
				} else {
					dir := os.Getenv(composeFile[1:end])
					project.DockerComposeFilenames[i] = path.Join(dir, composeFile[end:])
				}
			}
		}
	}

	// Find individual projects by locating docker-compose.yml files in the
	// specified project directories.  Create/synchronize a Project
	// configuration for each found Project.
	for _, projectDir := range devConfig.ProjectDirectories {
		// see if there is a project configuration
		getOrCreateProjectConfig(devConfig, projectDir)
	}

	for _, projectConfig := range devConfig.Projects {
		expandProject(devConfig, projectConfig)
	}
}
