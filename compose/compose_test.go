package compose_test

import (
	"strings"
	"testing"

	"github.com/wish/dev/compose"
	"github.com/wish/dev/test"

	"github.com/spf13/afero"
)

func TestConfigParseNoSuchFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := compose.Parse(fs, "/home/app", "does-not-exist.yml")
	if err == nil {
		t.Error("Expected error attempting to parse non-existent file")
	}
}

func TestConfigParseError(t *testing.T) {
	fs := afero.NewMemMapFs()

	test.CreateConfigFile(fs, "#!/usr/bin/python", "/home/app/foo.py")

	_, err := compose.Parse(fs, "/home/app", "/home/app/foo.py")
	if err == nil {
		t.Errorf("Expected error parsing file but got %s", err)
	} else if strings.Index(err.Error(), "Error parsing") == -1 {
		t.Errorf("Expected error parsing file but got %s", err)
	}
}

func TestConfigLoadError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// throw it the wrong type of file, make sure it fails..it's yaml
	// so will parse
	test.CreateConfigFile(fs, test.BigCoConfig, "/home/app/.dev.yaml")

	_, err := compose.Parse(fs, "/home/app", "/home/app/.dev.yaml")
	if err == nil {
		t.Errorf("Expected error loading file but got %s", err)
	} else if strings.Index(err.Error(), "Error loading") == -1 {
		t.Errorf("Expected error parsing file but got %s", err)
	}
}

func TestConfigLoadSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()

	test.CreateConfigFile(fs, test.AppCompose, "/home/app/docker-compose.yaml")

	config, err := compose.Parse(fs, "/home/app", "/home/app/docker-compose.yaml")
	if err != nil {
		t.Errorf("Not expecting an error but got %s", err)
	}
	if config == nil {
		t.Errorf("Not expecting a nil config")
	}
}
