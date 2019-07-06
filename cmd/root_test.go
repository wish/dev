package cmd

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/wish/dev"
	"github.com/wish/dev/test"

	"github.com/spf13/afero"
	"gotest.tools/env"
)

func TestInitializeWithoutDockerComposeInstalled(t *testing.T) {
	defer env.Patch(t, "DEV_CONFIG", "/home/test/.dev.yaml")()
	defer env.Patch(t, "PATH", "/usr/bin:/usr/local/bin:/sbin")()

	appConfig.SetFs(afero.NewMemMapFs())
	test.CreateConfigFile(appConfig.GetFs(), test.BigCoConfig, "/home/test/.dev.yaml")

	var calledOnFatalError bool
	logrus.StandardLogger().ExitFunc = func(x int) {
		calledOnFatalError = true
	}

	Initialize()

	if !calledOnFatalError {
		t.Error("Expected a fatal error but did not get one")
	}
}

func TestInitializeWithDevConfigSet(t *testing.T) {
	defer env.Patch(t, "DEV_CONFIG", "/home/test/.dev.yaml")()
	defer env.Patch(t, "PATH", "/usr/bin:/usr/local/bin:/sbin")()
	reset()

	appConfig.SetFs(afero.NewMemMapFs())
	test.CreateDockerCompose(appConfig.GetFs(), "/usr/local/bin")
	test.CreateConfigFile(appConfig.GetFs(), test.BigCoConfig, "/home/test/.dev.yaml")

	Initialize()

	var cmdTests = []struct {
		ProjectName string
		Aliases     []string
	}{
		{"postgresql", []string{"pg", "db"}},
		{"frontend", []string{"shiny"}},
	}

	for _, test := range cmdTests {
		cmd, _, err := rootCmd.Find([]string{test.ProjectName})
		if err != nil {
			t.Errorf("Expected to find '%s' project but got err: '%s'", test.ProjectName, err)
		}
		if cmd == nil {
			t.Errorf("Expected to find '%s' cmd, but got nil", test.ProjectName)
		}
		if cmd.Use != test.ProjectName {
			t.Errorf("Expected cmd to be named '%s', but got '%s'", test.ProjectName, cmd.Short)
		}
		if len(test.Aliases) != len(cmd.Aliases) {
			t.Errorf("Expected to find %d %s aliases, but got %d", len(test.Aliases), test.ProjectName,
				len(cmd.Aliases))
		}
		for _, alias := range test.Aliases {
			if !dev.SliceContainsString(cmd.Aliases, alias) {
				t.Errorf("Expected to find alias '%s' of '%s' cmd", alias, test.ProjectName)
			}
		}

		subCommands := []string{dev.UP, dev.BUILD, dev.PS, dev.SH, dev.UP}
		for _, subCmd := range subCommands {
			sCmd, _, err := cmd.Find([]string{subCmd})
			if err != nil {
				t.Errorf("Expected to find subcommand '%s' of %s but got err: '%s'", subCmd, test.ProjectName, err)
			}
			if sCmd == nil {
				t.Errorf("Expected to find '%s' sub-command of %s, but got nil", subCmd, test.ProjectName)
			}

			if sCmd.Use != subCmd {
				t.Errorf("Expected cmd to be named '%s', but got '%s'", subCmd, sCmd.Short)
			}
		}
	}
}

func TestInitializeWithoutDevConfigSet(t *testing.T) {
	homedir := "/home/test"
	defer env.Patch(t, "PATH", "/usr/bin:/usr/local/bin:/sbin")()
	defer env.Patch(t, "DEV_CONFIG", "")() // set to nothing so i can test locally where I set it
	defer env.Patch(t, "HOME", homedir)()

	reset()
	appConfig.SetFs(afero.NewMemMapFs())
	test.CreateDockerCompose(appConfig.GetFs(), "/usr/local/bin")
	test.CreateConfigFile(appConfig.GetFs(), test.BigCoConfig, homedir+"/.config/dev/dev.yaml")

	Initialize()

	var cmdTests = []struct {
		ProjectName string
		Aliases     []string
	}{
		{"postgresql", []string{"pg", "db"}},
		{"frontend", []string{"shiny"}},
	}

	for _, test := range cmdTests {
		cmd, _, err := rootCmd.Find([]string{test.ProjectName})
		if err != nil {
			t.Errorf("Expected to find '%s' project but got err: '%s'", test.ProjectName, err)
		}
		if cmd == nil {
			t.Errorf("Expected to find '%s' cmd, but got nil", test.ProjectName)
		}
		if cmd.Use != test.ProjectName {
			t.Errorf("Expected cmd to be named '%s', but got '%s'", test.ProjectName, cmd.Short)
		}
		if len(test.Aliases) != len(cmd.Aliases) {
			t.Errorf("Expected to find %d %s aliases, but got %d", len(test.Aliases), test.ProjectName,
				len(cmd.Aliases))
		}
		for _, alias := range test.Aliases {
			if !dev.SliceContainsString(cmd.Aliases, alias) {
				t.Errorf("Expected to find alias '%s' of '%s' cmd", alias, test.ProjectName)
			}
		}

		subCommands := []string{dev.UP, dev.BUILD, dev.PS, dev.SH, dev.UP}
		for _, subCmd := range subCommands {
			sCmd, _, err := cmd.Find([]string{subCmd})
			if err != nil {
				t.Errorf("Expected to find subcommand '%s' of %s but got err: '%s'", subCmd, test.ProjectName, err)
			}
			if sCmd == nil {
				t.Errorf("Expected to find '%s' sub-command of %s, but got nil", subCmd, test.ProjectName)
			}

			if sCmd.Use != subCmd {
				t.Errorf("Expected cmd to be named '%s', but got '%s'", subCmd, sCmd.Short)
			}
		}
	}
}

func TestInitializeWithoutConfig(t *testing.T) {
	homedir := "/home/test"
	defer env.Patch(t, "PATH", "/usr/bin:/usr/local/bin:/sbin")()
	defer env.Patch(t, "DEV_CONFIG", "")()
	defer env.Patch(t, "HOME", homedir)()

	reset()
	appConfig.SetFs(afero.NewMemMapFs())
	test.CreateDockerCompose(appConfig.GetFs(), "/usr/local/bin")

	Initialize()

	numCommands := len(rootCmd.Commands())
	if numCommands != 0 {
		t.Errorf("Expected 0 commands without a config, but got %d", numCommands)
	}
	for _, cmd := range rootCmd.Commands() {
		fmt.Printf("cmd %s", cmd.Use)
	}
}
