package registry

import (
	"bytes"
	"context"
	log "github.com/sirupsen/logrus"
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

	err := command.Run()

	if ctx.Err() != nil {
		log.Println("Timeout during docker login. If on Mac, CONSIDER UNLOCKING THE KEYCHAIN. Run")
		log.Println("  security -v unlock-keychain ~/Library/Keychains/login.keychain-db")
	}
	return err
}
