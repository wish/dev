package dev_test

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/spf13/afero"
	"github.com/wish/dev"
	"github.com/wish/dev/cmd"
	"github.com/wish/dev/config"
	"github.com/wish/dev/test"
	"gotest.tools/env"
)

var orderCalled int

type MockDep struct {
	Name           string
	Type           string
	Order          int
	ProjectConfig  *config.Project
	RegistryConfig *config.Registry
	NetworkConfig  *types.NetworkCreate
}

func (md *MockDep) PreRun(command string, appConfig *config.Dev, project *dev.Project) {
	md.Order = orderCalled
	orderCalled++
}

func (md *MockDep) Dependencies() []string {
	switch md.Type {
	case "network":
		return []string{}
	case "registry":
		return []string{}
	case "project":
		return md.ProjectConfig.Dependencies
	case "default":
		fmt.Println("unsupported dependency type")
	}
	return []string{}
}

func (md *MockDep) GetName() string {
	return md.Name
}

func createObjectMap(devConfig *config.Dev, t *testing.T) (map[string]dev.Dependency, map[string]*MockDep) {
	objMap := make(map[string]dev.Dependency)
	depMap := make(map[string]*MockDep)

	for name, opts := range devConfig.Projects {
		dep := &MockDep{Name: name, Type: "project", ProjectConfig: opts, Order: -1}
		objMap[name] = dep
		depMap[name] = dep

	}

	for name, opts := range devConfig.Networks {
		dep := &MockDep{Name: name, Type: "network", NetworkConfig: opts, Order: -1}
		objMap[name] = dep
		depMap[name] = dep
	}

	for name, opts := range devConfig.Registries {
		dep := &MockDep{Name: name, Type: "registry", RegistryConfig: opts, Order: -1}
		objMap[name] = dep
		depMap[name] = dep
	}

	return objMap, depMap
}

func TestInitDeps(t *testing.T) {
	defer env.Patch(t, "DEV_CONFIG", "/home/test/.dev.yaml")()
	defer env.Patch(t, "PATH", "/usr/bin:/usr/local/bin:/sbin")()
	cmd.Reset()

	cmd.AppConfig.SetFs(afero.NewMemMapFs())
	test.CreateDockerComposeBinary(cmd.AppConfig.GetFs(), "/usr/local/bin")
	test.CreateConfigFile(cmd.AppConfig.GetFs(), test.SmallCoConfig, "/home/test/.dev.yaml")

	cmd.Initialize()
	objMap, depMap := createObjectMap(cmd.AppConfig, t)

	proj := dev.NewProject(cmd.AppConfig.Projects["postgresql"])
	dev.InitDeps(objMap, cmd.AppConfig, "UP", proj)

	sharedDep := depMap["shared"]
	if sharedDep.Order == -1 {
		t.Errorf("shared dependency not initialized error")
	}
	if sharedDep.Order != 2 {
		t.Errorf("Expected order of initialization of shared to be 2, but got %d", sharedDep.Order)
	}

	appNet := depMap["app-net"]
	if appNet.Order == -1 {
		t.Errorf("app-net dependency not initialized error")
	}
	if appNet.Order > 1 {
		t.Errorf("Expected order of initialization of app-net to be <= 1 , but got %d", appNet.Order)
	}

	ecr := depMap["ecr"]
	if ecr.Order == -1 {
		t.Errorf("app-net dependency not initialized error")
	}
	if ecr.Order > 1 {
		t.Errorf("Expected order of initialization of ecr to be <= 1 , but got %d", ecr.Order)
	}
}
