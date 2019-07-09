package dev

import (
	"fmt"
	"os"
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

//type ProjectConstructor func(config *c.Project) *Project

// NewProject is the Project constructor.
func NewProject(config *c.Project) *Project {
	return &Project{
		Name:   config.Name,
		Config: config,
	}
}

// PreRun implements the Dependency interface. It brings up the project prior
// to the shell and up commads.
func (p *Project) PreRun(command string, appConfig *c.Dev, project *Project) {
	if !SliceContainsString([]string{UP, SH}, command) {
		return
	}

	p.Up(appConfig)
}

// Dependencies implements the Dependency interface. It returns a list of
// the names of its dependencies. These can be names of other projects,
// networks or registries.
func (p *Project) Dependencies() []string {
	return p.Config.Dependencies
}

// GetName returns the name of the project as configured by the user in the dev
// configuration file.
func (p *Project) GetName() string {
	return p.Name
}

// Up brings up the specified project container with its dependencies.
func (p *Project) Up(appConfig *c.Dev) {
	RunComposeUp(appConfig.ImagePrefix, p.Config.DockerComposeFilenames, "-d")
}

// UpFollowProjectLogs brings up the specified project with its dependencies
// and tails the logs of the project container.
func (p *Project) UpFollowProjectLogs(appConfig *c.Dev) {
	p.Up(appConfig)
	RunComposeLogs(appConfig.ImagePrefix, p.Config.DockerComposeFilenames, "-f", p.Config.Name)
}

// Shell runs commands or creates an interfactive shell on the Project
// container.
func (p *Project) Shell(appConfig *c.Dev, args []string) {
	running, err := docker.IsContainerRunning(p.Config.Name)
	if err != nil {
		log.Fatalf("Error communicating with docker daemon, is it up? %s", err)
	}
	if !running {
		log.Infof("Project %s not running, bringing it up", p.Config.Name)
		p.Up(appConfig)
	}

	// Get current directory, attempt to find its location
	// on the container and cd to it. This allows developers to
	// use relative directories like they would in a non-containerized
	// development environment.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %s", err)
	}

	relativePath := ""
	configDir := p.Config.Directory
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
		args = append(args, p.Config.Shell)
	}

	cmdLine := []string{p.Config.Shell, "-c",
		fmt.Sprintf("cd %s ; %s", relativePath, strings.Join(args, " "))}

	RunOnContainer(p.Name, cmdLine...)
}
