package dev

import (
	c "github.com/wish/dev/config"
)

const (
	// BUILD constant referring to the build command of this project which
	// builds the project with docker-compose as specified in this tools
	// configuration file.
	BUILD = "build"
	// DOWN constant referring to the "down" command of this project which
	// stops and removes the project container.
	DOWN = "down"
	// PS constant referring to the "ps" command of this project which shows
	// the status of the containers used by the project.
	PS = "ps"
	// SH constant referring to the "sh" command of this project which runs
	// commands on the project container.
	SH = "sh"
	// UP constant referring to the "up" command of this project which starts
	// the project and any of the specified dependencies.
	UP = "up"
)

// Dependency is the interface that is used by all objects in the dev
// configuration implement such that they can be used as a dependency by other
// objects of the configuration.
type Dependency interface {
	// PreRun does whatever is required of the dependency. It is run prior
	// to the specified command for the given project.
	PreRun(command string, appConfig *c.Dev, project *c.Project)
}
