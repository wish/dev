package dev

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	c "github.com/wish/dev/config"
	"github.com/wish/dev/registry"
)

// Registry is a private container registry that dev will attempt to login to
// so images can be pulled from it.
type Registry struct {
	Config *c.Registry
}

// NewRegistry constructs Registry objects.
func NewRegistry(config *c.Registry) *Registry {
	return &Registry{
		Config: config,
	}
}

// PreRun implements the Dependency interface.
func (r *Registry) PreRun(command string, appConfig *c.Dev, project *Project) {
	if !SliceContainsString([]string{BUILD, UP}, command) {
		return
	}

	err := registry.Login(r.Config.URL, r.Config.Name, r.Config.Password)
	if err != nil {
		msg := fmt.Sprintf("Failed to login to %s registry: %s", r.Config.Name, err)
		if r.Config.ContinueOnFailure {
			log.Warn(msg)
		} else {
			log.Fatal(msg)
		}
	} else {
		log.Debugf("Logged in to registry %s at %s", r.Config.Name, r.Config.URL)
	}
}

// Dependencies implements the Dependency interface.
func (r *Registry) Dependencies() []string {
	return []string{}
}

// GetName returns the name of the registry as defined by the user in the dev
// configuration file.
func (r *Registry) GetName() string {
	return r.Config.Name
}
