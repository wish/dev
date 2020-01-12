package dev_test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/wish/dev"
	"github.com/wish/dev/cmd"
	"github.com/wish/dev/test"
	"gotest.tools/env"
	"testing"
)

func TestCreateBuildableServiceList(t *testing.T) {
	defer env.Patch(t, "DEV_CONFIG", "/home/test/.dev.yaml")()
	defer env.Patch(t, "PATH", "/usr/bin:/usr/local/bin:/sbin")()
	cmd.Reset()

	cmd.AppConfig.SetFs(afero.NewMemMapFs())
	test.CreateDockerComposeBinary(cmd.AppConfig.GetFs(), "/usr/local/bin")
	test.CreateConfigFile(cmd.AppConfig.GetFs(), test.SmallCoConfig, "/home/test/.dev.yaml")
	test.CreateConfigFile(cmd.AppConfig.GetFs(), test.AppCompose, "/home/test/db.docker-compose.yml")

	cmd.Initialize()
	proj := dev.NewProject(cmd.AppConfig.Projects["postgresql"])
	got := dev.CreateBuildableServiceList(cmd.AppConfig, proj.Config)

	want := []string{"app"}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CreateBuildableServiceList() mismatch (-want +got):\n%s", diff)
	}
}
