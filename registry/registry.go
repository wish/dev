package registry

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// Login attempts to perform a user/password login to the registry provided.
// If unable to login an error is returned, otherwise nil is returned.
func Login(URL, username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	command := exec.CommandContext(ctx, "docker", "login", URL,
		"--username", username, "--password-stdin")
	command.Stdin = bytes.NewBuffer([]byte(password))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
