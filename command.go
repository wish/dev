package dev

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
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
func runDockerCompose(cmd, project string, composePaths []string, args ...string) {
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

// RunComposeBuild runs docker-compose build with the specified docker compose
// files and args.
func RunComposeBuild(project string, composePaths []string, args ...string) {
	runDockerCompose("build", project, composePaths, args...)
}

// RunComposeUp runs docker-compose up with the specified docker compose
// files and args.
func RunComposeUp(project string, composePaths []string, args ...string) {
	runDockerCompose("up", project, composePaths, args...)
}

// RunComposePs runs docker-compose ps with the specified docker compose
// files and args.
func RunComposePs(project string, composePaths []string, args ...string) {
	runDockerCompose("ps", project, composePaths, args...)
}

// RunComposeLogs runs docker-compose logs with the specified docker compose
// files and args.
func RunComposeLogs(project string, composePaths []string, args ...string) {
	runDockerCompose("logs", project, composePaths, args...)
}

// RunComposeDown runs docker-compose down with the specified docker compose
// files and args.
func RunComposeDown(project string, composePaths []string, args ...string) {
	runDockerCompose("down", project, composePaths, args...)
}

// RunOnContainer runs the commands on the container with the specified
// name using the 'docker' command.
func RunOnContainer(containerName string, cmds ...string) {
	cmdLine := []string{"exec"}

	// avoid "input device is not a tty error"
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmdLine = append(cmdLine, "-it")
	}

	cmdLine = append(cmdLine, containerName)

	for _, cmd := range cmds {
		cmdLine = append(cmdLine, cmd)
	}

	runCommand("docker", cmdLine)
}
