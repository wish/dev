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

// NetworkCreate sends a request to the local docker daemon to create the ipam
// network specified with name with the provided ipam options.
func NetworkCreate(name string, options *types.NetworkCreate) error {
	if os.Getenv("DOCKER_API_VERSION") == "" {
		// Bump this as testing is performed on later versions
		os.Setenv("DOCKER_API_VERSION", "1.25")
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		errors.Wrap(err, "failed to create docker client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSecondsDefault)
	defer cancel()

	res, err := cli.NetworkCreate(ctx, name, *options)
	if err != nil {
		errors.Wrap(err, "failed to create network")
		return err
	}
	if res.Warning != "" {
		log.Warn(res.Warning)
	}
	return err

}

// func NetworkExists() {
// }

// func NetworkConfigMatches() {
// }

// func NetworkConfigModify() {
// }

// func NetworkDestroy() {
// }
