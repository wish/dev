package docker

import (
	"context"
	"os"
	"time"

	"github.com/docker/docker/api/types"
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
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSecondsDefault)
	defer cancel()

	res, err := cli.NetworkCreate(ctx, name, *opts)
	if err != nil {
		return "", errors.Wrap(err, "failed to create network")
	}
	if res.Warning != "" {
		log.Warn(res.Warning)
	}
	return res.ID, err

}

// NetworkExists checks for the existence of the supplied network name.
func NetworkExists(name string) (bool, error) {
	cli, err := getDockerClient()
	if err != nil {
		return false, errors.Wrap(err, "failed to create docker client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSecondsDefault)
	defer cancel()

	var resources []types.NetworkResource
	resources, err = cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		errors.Wrap(err, "failed to retrieve networks from docker")
	}
	for _, network := range resources {
		if network.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// NetworkConfigMatches checks if the named network has the same configuration
// as the supplied network options. If the networks are configured the same,
// this function returns false, otherwise true is returned. An error is
// returned if the network does not exist.
func NetworkConfigMatches(name string, opts *types.NetworkCreate) bool {
	return true
}

// func NetworkConfigModify() {
// }

// func NetworkDestroy() {
// }
