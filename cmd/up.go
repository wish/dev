package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wish/dev"
	"github.com/wish/dev/compose"
	"github.com/wish/dev/docker"
	"github.com/wish/dev/registry"
)

// networksCreate creates any external network configured in the dev tool if
// it does not exist already. It returns a map from name to the network id
// of all the external networks.
func networksCreate(config *dev.Config) map[string]string {
	networkIDMap := make(map[string]string, len(config.Networks))
	for name, opts := range config.Networks {
		networkID, err := docker.NetworkIDFromName(name)
		if err != nil {
			err = errors.Wrapf(err, "Error checking if network %s exists", name)
			log.Fatal(err)
		}

		if networkID == "" {
			networkID, err = docker.NetworkCreate(name, opts)
			log.Infof("Created %s network %s", name, networkID)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Debugf("Network %s already exists with id %s", name, networkID)
		}
		networkIDMap[name] = networkID
	}
	return networkIDMap
}

// registriesLogin logs in to the specified registries. So we can fetch from
// private registries.
func registriesLogin(config *dev.Config) {
	for _, r := range config.Registries {
		err := registry.Login(r.URL, r.Name, r.Password)
		if err != nil {
			msg := fmt.Sprintf("Failed to login to %s registry: %s", r.Name, err)
			if r.ContinueOnFailure {
				log.Warn(msg)
			} else {
				log.Fatal(msg)
			}
		} else {
			log.Debugf("Logged in to registry %s at %s", r.Name, r.URL)
		}
	}
}

// createNetworkServiceMap creates a mapping from the networks configured by dev
// to a list of the services that use them in the projects docker-compose files.
func createNetworkServiceMap(devConfig *dev.Config, project *dev.Project,
	networkIDMap map[string]string) map[string][]string {
	serviceNetworkMap := make(map[string][]string, len(devConfig.Networks))
	for _, composeFilename := range project.DockerComposeFilenames {
		composeConfig, err := compose.Parse(project.Directory, composeFilename)
		if err != nil {
			log.Fatal("Failed to parse docker-compose config file: ", err)
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
// containers into the state specified in the dev config files.
//
// Networks do not persist reboots. Container configured with an old network id
// that no longer exists will not be able to start (docker-compose up will fail
// when it attempts to start the container). These containers must be removed
// before we attempt to start the container.
func verifyContainerConfig(config *dev.Config, project *dev.Project, networkIDMap map[string]string) {
	if len(networkIDMap) == 0 {
		// no external networks, nothing to do
		return
	}

	networkServiceMap := createNetworkServiceMap(config, project, networkIDMap)
	for networkName, services := range networkServiceMap {
		networkID := networkIDMap[networkName]
		err := docker.RemoveContainerIfRequired(networkName, networkID, services)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// ProjectCmdUpCreate constructs the 'up' command line option available for
// each project.
func ProjectCmdUpCreate(config *dev.Config, project *dev.Project) *cobra.Command {
	up := &cobra.Command{
		Use:   "up",
		Short: "Create and start the " + project.Name + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			registriesLogin(config)
			networkIDMap := networksCreate(config)
			verifyContainerConfig(config, project, networkIDMap)

			runDockerCompose(config.ImagePrefix, "up", project.DockerComposeFilenames, "-d")
			runDockerCompose(config.ImagePrefix, "logs", project.DockerComposeFilenames, "-f", project.Name)
		},
	}
	return up
}
