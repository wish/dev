package docker

import (
	"context"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	timeoutSecondsDefault = time.Second * 2
)

func getDockerClient() (*client.Client, error) {
	if os.Getenv("DOCKER_API_VERSION") == "" {
		// Bump this as testing is performed on later versions
		if err := os.Setenv("DOCKER_API_VERSION", "1.25"); err != nil {
			log.Warn("Failed to set DOCKER_API_VERSION")
		}
	}
	return client.NewEnvClient()
}

// NetworkCreate sends a request to the local docker daemon to create the ipam
// network specified with name with the provided ipam options.
//
// Returns the network id of the created network or an error.
func NetworkCreate(name string, opts *types.NetworkCreate) (string, error) {
	if os.Getenv("DOCKER_API_VERSION") == "" {
		// Bump this as testing is performed on later versions
		if err := os.Setenv("DOCKER_API_VERSION", "1.25"); err != nil {
			log.Warn("Failed to set DOCKER_API_VERSION")
		}
	}
	cli, err := getDockerClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to create docker client")
	}
	res, err := cli.NetworkCreate(context.Background(), name, *opts)
	if err != nil {
		return "", errors.Wrap(err, "failed to create network")
	}
	if res.Warning != "" {
		log.Warn(res.Warning)
	}
	return res.ID, err

}

// NetworkIDFromName checks for the existence of the supplied network name and
// returns its id if it exists. If it does not exist an empty string is
// returned.
func NetworkIDFromName(name string) (string, error) {
	cli, err := getDockerClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to create docker client")
	}

	var resources []types.NetworkResource // saving you 20 seconds since 2019
	resources, err = cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve networks from docker")
	}
	for _, network := range resources {
		if network.Name == name {
			return network.ID, nil
		}
	}
	return "", nil
}

// RemoveContainerIfRequired checks each exited container in the container name list and
// removes any of those containers if it is attached to the network name provided but with
// a different network ID.
func RemoveContainerIfRequired(networkName, networkID string, containerNames []string) error {
	cli, err := getDockerClient()
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}

	// only protecting against containers connected to the dead/wrong network,
	// so we only need to look for stopped containers here. Might be nice
	// to verify the network settings are correct/haven't changed for up
	// containers..
	options := types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("status", "exited")),
	}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		return err
	}

	containerMap := make(map[string]bool, len(containerNames))
	for _, name := range containerNames {
		containerMap[name] = true
	}

	for _, container := range containers {
		modNames := make([]string, len(container.Names))
		for i, name := range container.Names {
			// why are slashes added to container names?
			if string(name[0]) == "/" {
				modNames[i] = name[1:]
			} else {
				modNames[i] = name
			}
		}

		for _, name := range modNames {
			if _, ok := containerMap[name]; ok {
				// check if this container is attached to the
				// specified network but has a different id for
				// it. If so, remove the container as it cannot
				// be brought up with a network that doesn't
				// exist..mostly likely to happen on reboot.
				if settings, ok := container.NetworkSettings.Networks[networkName]; ok {
					if settings.NetworkID != networkID {
						log.Debugf("%s attached to %s with a different network id, removing", name, networkName)
						opts := types.ContainerRemoveOptions{}
						if err := cli.ContainerRemove(context.Background(), container.ID, opts); err != nil {
							return err
						}
					}
				}
			}
		}

	}
	return nil
}

// IsContainerRunning checks if there is a container with status "up" for the
// specified project.
func IsContainerRunning(name string) (bool, error) {
	cli, err := getDockerClient()
	if err != nil {
		return false, errors.Wrap(err, "failed to create docker client")
	}

	options := types.ContainerListOptions{
		//All: true,
		Filters: filters.NewArgs(
			filters.Arg("status", "running"),
			filters.Arg("name", name)),
	}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		return false, errors.Wrap(err, "Failed to check container status")
	}
	//log.Debugf("containers: %+v", containers)
	return len(containers) > 0, nil
}
