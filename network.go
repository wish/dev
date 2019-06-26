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

// NewNetwork is the Network constructor.
func NewNetwork(name string, config *types.NetworkCreate) *Network {
	return &Network{
		Name:   name,
		Config: config,
	}
}

// create any external network configured in the dev tool if it does not exist
// already. It returns the network id used to indentify the network by docker.
func (n *Network) create() string {
	networkID, err := docker.NetworkIDFromName(n.Name)
	if err != nil {
		err = errors.Wrapf(err, "Error checking if network %s exists", n.Name)
		log.Fatal(err)
	}
	if networkID == "" {
		networkID, err = docker.NetworkCreate(n.Name, n.Config)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Created %s network %s", n.Name, networkID)
	} else {
		log.Debugf("Network %s already exists with id %s", n.Name, networkID)
	}

	return networkID
}

// createNetworkServiceMap creates a mapping from the networks configured by dev
// to a list of the services that use them in the projects docker-compose files.
func (n *Network) createNetworkServiceMap(devConfig *config.Dev, project *config.Project,
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

// verifyContainerConfig performs container operations necessary to get the
// containers into the state specified in the dev appConfig files.
//
// Networks do not persist reboots. Container configured with an old network id
// that no longer exists will not be able to start (docker-compose up will fail
// when it attempts to start the container). These containers must be removed
// before we attempt to start the container.
func (n *Network) verifyContainerConfig(appConfig *config.Dev, project *config.Project, networkID string) {
	networkIDMap := map[string]string{
		n.Name: networkID,
	}

	networkServiceMap := n.createNetworkServiceMap(appConfig, project, networkIDMap)
	for networkName, services := range networkServiceMap {
		networkID := networkIDMap[networkName]
		err := docker.RemoveContainerIfRequired(networkName, networkID, services)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// PreRun implements the Dependency interface. It will destroy any containers
// that are attached to a no longer existing network of the same name such that
// the containers can be created with the correct network.
func (n *Network) PreRun(command string, appConfig *c.Dev, project *Project) {
	if !SliceContainsString([]string{UP, SH}, command) {
		return
	}
	networkID := n.create()
	n.verifyContainerConfig(appConfig, project.Config, networkID)
}

// Dependencies implements the Dependency interface.  At this time a Network
// cannot have dependencies so it returns an empty slice.
func (n *Network) Dependencies() []string {
	return []string{}
}

// GetName returns the name of the network as named by the user in the
// configuration file.
func (n *Network) GetName() string {
	return n.Name
}
