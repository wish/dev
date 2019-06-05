package dev

import (
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/wish/dev/compose"
	"github.com/wish/dev/config"
	c "github.com/wish/dev/config"
	"github.com/wish/dev/docker"
)

// Network is an external docker network that dev manages.
type Network struct {
	Name   string
	Config *types.NetworkCreate
}

// NewNetwork is a Network constructor.
func NewNetwork(name string, config *types.NetworkCreate) *Network {
	return &Network{
		Name:   name,
		Config: config,
	}
}

// networksCreate creates any external network configured in the dev tool if
// it does not exist already. It returns the network id used to indentify the
// network by docker.
func (r *Network) create() string {
	networkID, err := docker.NetworkIDFromName(r.Name)
	if err != nil {
		err = errors.Wrapf(err, "Error checking if network %s exists", r.Name)
		log.Fatal(err)
	}
	if networkID == "" {
		networkID, err = docker.NetworkCreate(r.Name, r.Config)
		log.Infof("Created %s network %s", r.Name, networkID)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Debugf("Network %s already exists with id %s", r.Name, networkID)
	}

	return networkID
}

// createNetworkServiceMap creates a mapping from the networks configured by dev
// to a list of the services that use them in the projects docker-compose files.
func (r *Network) createNetworkServiceMap(devConfig *config.Dev, project *config.Project,
	networkIDMap map[string]string) map[string][]string {

	serviceNetworkMap := make(map[string][]string, len(networkIDMap))
	for _, composeFilename := range project.DockerComposeFilenames {
		composeConfig, err := compose.Parse(project.Directory, composeFilename)
		if err != nil {
			log.Fatal("Failed to parse docker-compose appConfig file: ", err)
		}

		for _, service := range composeConfig.Services {
			for name := range service.Networks {
				if _, ok := networkIDMap[name]; ok {
					serviceNetworkMap[name] = append(serviceNetworkMap[name], service.Name)
				}
			}
		}
	}
	return serviceNetworkMap
}

// updateContainers performs container operations necessary to get the
// containers into the state specified in the dev appConfig files.
//
// Networks do not persist reboots. Container configured with an old network id
// that no longer exists will not be able to start (docker-compose up will fail
// when it attempts to start the container). These containers must be removed
// before we attempt to start the container.
func (r *Network) verifyContainerConfig(appConfig *config.Dev, project *config.Project, networkID string) {
	networkIDMap := map[string]string{
		r.Name: networkID,
	}

	networkServiceMap := r.createNetworkServiceMap(appConfig, project, networkIDMap)
	for networkName, services := range networkServiceMap {
		networkID := networkIDMap[networkName]
		err := docker.RemoveContainerIfRequired(networkName, networkID, services)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// PreRun implements the Dependency interface. It will destroy any containers
// that are attached to a no longer existing networ of the same name such that
// the containers can be created with the correct network.
func (r *Network) PreRun(command string, appConfig *c.Dev, project *c.Project) {
	if !SliceContainsString([]string{UP, SH}, command) {
		return
	}
	networkID := r.create()
	r.verifyContainerConfig(appConfig, project, networkID)
}
