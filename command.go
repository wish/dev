package dev

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/wish/dev/config"
)

// A dumping ground for utilities used across commands that use exec.Command to run..

func runCommand(name string, args []string) {
	log.Debugf("Running: %s %s", name, strings.Join(args, " "))
	command := exec.Command(name, args...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	command.Run()
}

// RunDockerCompose runs docker-compose wih the specified subcommand and
// arguments.
func RunDockerCompose(cmd, project string, composePaths []string, args ...string) {
	cmdLine := []string{"-p", project}

	for _, path := range composePaths {
		cmdLine = append(cmdLine, "-f", path)
	}
	// one of build, exec, etc.
	cmdLine = append(cmdLine, cmd)

	// append any additional arguments or flags, i.e., -d
	for _, arg := range args {
		cmdLine = append(cmdLine, arg)
	}

	runCommand("docker-compose", cmdLine)
}

// RunOnContainer runs the commands on the Project container specified in
// config.Project using the docker command.
func RunOnContainer(projectName string, project *config.Project, cmds ...string) {
	cmdLine := []string{"-p", projectName}

	// avoid "input device is not a tty error"
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmdLine = append(cmdLine, "-it")
	}

	cmdLine = append(cmdLine, project.Name)

	for _, cmd := range cmds {
		cmdLine = append(cmdLine, cmd)
	}

	runCommand("docker", cmdLine)
}
