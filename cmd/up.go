package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wish/dev"
	"github.com/wish/dev/docker"
	"github.com/wish/dev/registry"
)

// ProjectCmdUpCreate constructs the 'up' command line option available for
// each project.
func ProjectCmdUpCreate(config *dev.Config, project *dev.Project) *cobra.Command {
	up := &cobra.Command{
		Use:   "up",
		Short: "Create and start the " + project.Name + " containers",
		Run: func(cmd *cobra.Command, args []string) {
			for _, r := range config.Registries {
				err := registry.Login(r)
				if err != nil {
					msg := fmt.Sprintf("Failed to login to %s registry: %s", r.Name, err)
					if r.ContinueOnFailure {
						log.Warn(msg)
					} else {
						log.Fatal(msg)
					}

				}
			}
			for name, opts := range config.Networks {
				log.Infof("Creating %s network", name)
				if err := docker.NetworkCreate(name, opts); err != nil {
					log.Fatal(err)
				}

			}
			runDockerCompose("up", project.DockerComposeFilenames, "-d")
			runDockerCompose("logs", project.DockerComposeFilenames, "-f", project.Name)
		},
	}
	return up
}
