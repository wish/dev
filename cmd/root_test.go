package cmd

import (
	"testing"

	"github.com/wish/dev/test"

	"github.com/spf13/afero"
	"gotest.tools/env"
)

func TestInitialize(t *testing.T) {
	defer env.Patch(t, "DEV_CONFIG", "/home/test/.dev.yaml")()
	AppConfig.SetFs(afero.NewMemMapFs())
	test.CreateConfigFile(AppConfig.GetFs(), test.BigCoConfig, "/home/test/.dev.yaml")

	Initialize()
}
