package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	c "github.com/wish/dev/config"
	"github.com/wish/dev/docker"
)

// Project is the group of functionality provided by a docker-compose file.
type Project struct {
	Name   string
	Config *c.Project
}

// NewProject is a Project constructor.
func NewProject(config *c.Project) *Project {
	return &Project{
		Name:   config.Name,
		Config: config,
	}
}

// PreRun implements the Dependency interface; will bring up the project prior
// to the shell and up commads.
func (r *Project) PreRun(command string, appConfig *c.Dev, project *c.Project) {
	if !SliceContainsString([]string{UP, SH}, command) {
		return
	}
	// handle dependencies for this project

	RunDockerCompose("up", appConfig.ImagePrefix, r.Config.DockerComposeFilenames, "-d")
}

// Up brings up the specified project with its dependencies and optionally
// tails the logs of the project container.
func (r *Project) Up(appConfig *c.Dev, followLogs bool) {
	// registry dependency

	// network dependency

	RunDockerCompose("up", appConfig.ImagePrefix, r.Config.DockerComposeFilenames, "-d")
	if followLogs {
		RunDockerCompose("logs", appConfig.ImagePrefix, r.Config.DockerComposeFilenames, "-f", r.Config.Name)
	}
}

// Shell runs commands or creates an interfactive shell on the Project
// container.
func (r *Project) Shell(appConfig *c.Dev, args []string) {
	running, err := docker.IsContainerRunning(appConfig.ImagePrefix, r.Config.Name)
	if err != nil {
		log.Fatalf("Error communicating with docker daemon, is it up? %s", err)
	}
	if !running {
		log.Infof("Project %s not running, bringing it up", r.Config.Name)
		r.Up(appConfig, false)
	}

	// Get current directory, attempt to find its location
	// on the container and cd to it. This allows developers to
	// use relative directories like they would in a non-containerized
	// development environment.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %s", err)
	}
	configDir := filepath.Dir(appConfig.Filename)

	relativePath := ""
	if strings.HasPrefix(cwd, configDir) {
		start := strings.Index(cwd, configDir) + len(configDir) + 1
		if start < len(cwd) {
			relativePath = cwd[start:]
		} else {
			relativePath = "."
		}
	}

	if len(args) == 0 {
		// no subcommands, so just provide a shell
		args = append(args, r.Config.Shell)
	}

	cmdLine := []string{r.Config.Shell, "-c",
		fmt.Sprintf("cd %s ; %s", relativePath, strings.Join(args, " "))}

	RunOnContainer(appConfig.ImagePrefix, r.Config, cmdLine...)
}
