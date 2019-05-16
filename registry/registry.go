package registry

import (
	"time"

	"github.com/wish/dev"
	"github.com/wish/docker-registry-client/registry"
)

// Login attempts to perform a user/password login to the registry provided.
// If unable to login an error is returned, otherwise nil is returned.
func Login(r *dev.Registry) error {
	timeout := time.Duration(2) * time.Second
	_, err := registry.New(r.URL, r.Username, r.Password, timeout, nil)
	return err
}
