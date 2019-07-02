package dev

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

// Commander is a wrapper around the exec.Command interface. It only contains
// what we need in order to substitue it with something else that works a
// little better for testing.
type Commander interface {
	Run() error
}

var cmdExecutor Commander = nil

type testCommander struct {
}

func (tc *testCommander) Run() error {
	return nil
}

func setExecutor(executor Commander) {
	cmdExecutor = executor
}

func getExecutor(name string, args ...string) Commander {
	if cmdExecutor == nil {
		cmd := exec.Command(name, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd
	}
	return cmdExecutor
}

func runCommand(name string, args []string) {
	log.Debugf("Running: %s %s", name, strings.Join(args, " "))
	command := getExecutor(name, args...)
	command.Run()
}

// RunDockerCompose runs docker-compose with the specified subcommand and
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
