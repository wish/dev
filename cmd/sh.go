package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wish/dev"
)

// ProjectCmdShCreate constructs the 'sh' command line option available for
// each project.
func ProjectCmdShCreate(config *dev.Config, project *dev.Project) *cobra.Command {
	// needs work here... to pass args, gotta quote everything... -- doesn't work, etc.
	sh := &cobra.Command{
		Use:   "sh",
		Short: "Get a shell on the " + project.Name + " container",
		Args:  cobra.ArbitraryArgs,
		// Need to handle the flags manually. We do this so that we can
		// send in flags to the container without quoting the entire
		// string-- in the name of usability.
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Get current directory, attempt to find its location
			// on the container and cd to it. This allows developers to
			// use relative directories like they would in a non-containerized
			// development environment.
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("Failed to get current directory: %s", err)
			}
			configDir := filepath.Dir(config.Filename)

			relativePath := ""
			if strings.HasPrefix(cwd, configDir) {
				start := strings.Index(cwd, configDir) + len(configDir) + 1
				if start < len(cwd) {
					relativePath = cwd[start:]
				} else {
					relativePath = "."
				}
			}

			if len(args) > 0 {
				// assume a command starting with a dash is a
				// cry for help. Make this smarter..
				if strings.HasPrefix(args[0], "-") {
					cmd.Help()
					return
				}
			} else {
				// no subcommands, so just provide a shell
				args = append(args, project.Shell)
			}

			cmdLine := []string{project.Shell, "-c",
				fmt.Sprintf("cd %s ; %s", relativePath, strings.Join(args, " "))}

			runOnContainer(project, cmdLine...)
		},
	}
	return sh
}
