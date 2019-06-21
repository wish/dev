package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const (
	BigCoDirName  = "/home/nobody"
	BigCoFilename = ".dev.yaml"
	BigCoFullPath = BigCoDirName + "/" + BigCoFilename
)

const BigCoConfig = `
log:
  level: "info"

image_prefix: "bigco"

projects:
  postgresql:
    aliases: ["db"]
    docker_compose_files:
      - "shared.docker-compose.yml"
      - "db.docker-compose.yml"
    depends_on: ["app-net", "dev-registry"]

  frontend:
    aliases: ["shiny"]
    docker_compose_files:
      - "shared.docker-compose.yml"
      - "frontend/docker-compose.yml"
    depends_on: ["app-net", "ecr"]

networks:
  app-net:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 173.16.242.0/16

registries:
    ecr:
      url: "https://aws.ecr.my-region"
      username: "developer"
      password: "procrastination"
      continue_on_failure: True
`

const dependentAppConfig = `
log:
  level: "debug"

image_prefix: "bigco"

projects:
  scraper:
    aliases: ["s"]
    docker_compose_files:
      - "docker-compose.yml"
    depends_on: ["app-net", "ecr"]
`

func newConfigReader(configStr string) *strings.Reader {
	return strings.NewReader(configStr)
}

func configFromString(configFilename, configStr string) *Dev {
	v := viper.New()
	v.SetConfigFile(configFilename) // so viper knows what kind of data to expect
	reader := newConfigReader(configStr)
	v.ReadConfig(reader)

	devConfig := NewConfig()
	v.Unmarshal(devConfig)
	return devConfig
}

func expandedConfigFromString(configFilename, configStr string) *Dev {
	c := configFromString(configFilename, configStr)
	Expand(configFilename, c)
	return c
}

func newBigCoConfig() *Dev {
	return configFromString(BigCoFullPath, BigCoConfig)
}

func TestIsDefaultConfig(t *testing.T) {
	devConfig := NewConfig()
	if !isDefaultConfig(devConfig) {
		t.Error("unmodified NewConfig != DefaultConfig")
	}

	devConfig.ImagePrefix = "bigco"
	if isDefaultConfig(devConfig) {
		t.Error("expected modified config after after changing image prefix")
	}
}

func TestExpandOneFileAndDir(t *testing.T) {
	devConfig := newBigCoConfig()

	Expand(BigCoFullPath, devConfig)

	if devConfig.Dir != BigCoDirName {
		t.Errorf("Expected config.Dir to be %s but got %s", BigCoDirName, devConfig.Dir)
	}

	if devConfig.Filename != BigCoFullPath {
		t.Errorf("Expected config.Filename to be %s but got %s", BigCoFilename, BigCoFullPath)
	}
}

func TestExpandRegistry(t *testing.T) {
	devConfig := newBigCoConfig()

	Expand(BigCoFullPath, devConfig)

	if devConfig.Filename != BigCoFullPath {
		t.Errorf("Expected config.Filename to be %s but got %s", BigCoFilename, BigCoFullPath)
	}

	numRegistries := len(devConfig.Registries)
	if numRegistries != 1 {
		t.Errorf("Expected to have one registry, but have %d", numRegistries)
	}
	registry, exists := devConfig.Registries["ecr"]
	if !exists {
		t.Error("Expected to find a registry named ecr")
	} else {
		if registry.Name != "ecr" {
			t.Errorf("Expected registry to be named ecr, but got %s", registry.Name)
		}
	}
}

func TestExpandProjects(t *testing.T) {
	devConfig := newBigCoConfig()

	Expand(BigCoFullPath, devConfig)

	numProjects := len(devConfig.Projects)
	if numProjects != 2 {
		t.Errorf("Expected to have 2 projects, but have %d", numProjects)
	}

	project, exists := devConfig.Projects["postgresql"]
	if exists {
		if project.Name != "postgresql" {
			t.Errorf("Expected project to be named postgresql, but got %s", project.Name)
		}
	} else {
		t.Error("Expected to find a registry named postgresql")
	}

	project, exists = devConfig.Projects["frontend"]
	if exists {
		if project.Name != "frontend" {
			t.Errorf("Expected project to be named frontend, but got %s", project.Name)
		}
	} else {
		t.Error("Expected to find a registry named frontend")
	}
}

func TestDefaultPrefixMatchesDirName(t *testing.T) {
	config := `
projects:
  scraper:
    docker_compose_files:
      - "docker-compose.yml"
`
	c := expandedConfigFromString("/home/scraper/dev.yaml", config)
	if c.ImagePrefix != "scraper" {
		t.Errorf("Expected image prefix to be 'scraper' but got '%s'", c.ImagePrefix)
	}
}

func TestMergeForceSamePrefix(t *testing.T) {
	config := `
projects:
  scraper:
    docker_compose_files:
      - "docker-compose.yml"
`
	c := expandedConfigFromString("/home/scraper/dev.yaml", config)
	target := NewConfig()
	Merge(target, c)

	devConfig := newBigCoConfig()
	Expand(BigCoFullPath, devConfig)
	err := Merge(c, devConfig)
	if err == nil {
		t.Errorf("Expected error from mismatched image prefix but got none")
	}
}

func TestMergeGlobalOptionsSetByFirstConfig(t *testing.T) {
	config := `
log:
  level: "debug"

image_prefix: "bigco"

projects:
  scraper:
    docker_compose_files:
      - "docker-compose.yml"
`
	c := expandedConfigFromString("/home/scraper/dev.yaml", config)
	target := NewConfig()
	err := Merge(target, c)
	if err != nil {
		t.Errorf("Unexpected failure of merge: %s", err)
	}
	if target.Log.Level != "debug" {
		t.Errorf("Expected log level of 'debug' but got '%s'", target.Log.Level)
	}
	if target.Dir != "/home/scraper" {
		t.Errorf("Expected target.Dir of '/home/scraper' but got '%s'", target.Dir)
	}
	project, exists := target.Projects["scraper"]
	if exists {
		fullPath := "/home/scraper/docker-compose.yml"
		if project.DockerComposeFilenames[0] != fullPath {
			t.Errorf("Expected project scraper to have docker compose file of '%s' but got '%s'", fullPath, project.DockerComposeFilenames[0])
		}
	} else {
		t.Errorf("Expected to find a 'scraper' project")
	}

	devConfig := newBigCoConfig()
	Expand(BigCoFullPath, devConfig)
	err = Merge(target, devConfig)
	if err != nil {
		t.Errorf("Unexpected failure of merge: %s", err)
	}
	if target.Log.Level != "debug" {
		t.Errorf("Expected log level of 'debug' but got '%s'", target.Log.Level)
	}
	if target.Dir != "/home/scraper" {
		t.Errorf("Expected target.Dir of '/home/scraper' but got '%s'", target.Dir)
	}

	project, exists = target.Projects["frontend"]
	if exists {
		fullPath := "/home/nobody/shared.docker-compose.yml"
		if project.DockerComposeFilenames[0] != fullPath {
			t.Errorf("Expected project frontend to have docker compose file of '%s' but got '%s'", fullPath, project.DockerComposeFilenames[0])
		}
		fullPath = "/home/nobody/frontend/docker-compose.yml"
		if project.DockerComposeFilenames[1] != fullPath {
			t.Errorf("Expected project frontend to have docker compose file of '%s' but got '%s'", fullPath, project.DockerComposeFilenames[1])
		}
	} else {
		t.Errorf("Expected to find a 'scraper' project")
	}
}

func TestMergeDuplicateProjectNames(t *testing.T) {
	config := `
log:
  level: "debug"

image_prefix: "bigco"

projects:
  frontend:
    docker_compose_files:
      - "docker-compose.yml"
`
	c := expandedConfigFromString("/home/scraper/dev.yaml", config)
	target := NewConfig()
	err := Merge(target, c)
	if err != nil {
		t.Errorf("Unexpected failure of merge: %s", err)
	}

	devConfig := newBigCoConfig()
	Expand(BigCoFullPath, devConfig)
	err = Merge(target, devConfig)
	if err == nil {
		t.Error("Expected duplicate project name error")
	} else {
		if strings.Index(err.Error(), "duplicate") == -1 {
			t.Errorf("Expected error string to include the word 'duplicate', but got '%s'", err.Error())
		}
	}
}

func TestMergeDuplicateNetworkNames(t *testing.T) {
	config := `
log:
  level: "debug"

image_prefix: "bigco"

projects:
  foo:
    docker_compose_files:
      - "docker-compose.yml"
networks:
  app-net:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 192.16.242.0/16
`
	c := expandedConfigFromString("/home/scraper/dev.yaml", config)
	target := NewConfig()
	err := Merge(target, c)
	if err != nil {
		t.Errorf("Unexpected failure of merge: %s", err)
	}

	devConfig := newBigCoConfig()
	Expand(BigCoFullPath, devConfig)
	err = Merge(target, devConfig)
	if err == nil {
		t.Error("Expected duplicate network name error")
	} else {
		if strings.Index(err.Error(), "duplicate") == -1 {
			t.Errorf("Expected error string to include the word 'duplicate', but got '%s'", err.Error())
		}
	}
}

func TestMergeDuplicateRegistryNames(t *testing.T) {
	config := `
log:
  level: "debug"

image_prefix: "bigco"

projects:
  foo:
    docker_compose_files:
      - "docker-compose.yml"

registries:
    ecr:
      url: "https://aws.ecr.my-region"
      username: "developer"
      password: "procrastination"
`
	c := expandedConfigFromString("/home/scraper/dev.yaml", config)
	target := NewConfig()
	err := Merge(target, c)
	if err != nil {
		t.Errorf("Unexpected failure of merge: %s", err)
	}

	devConfig := newBigCoConfig()
	Expand(BigCoFullPath, devConfig)
	err = Merge(target, devConfig)
	if err == nil {
		t.Error("Expected duplicate registry error")
	} else {
		if strings.Index(err.Error(), "duplicate") == -1 {
			t.Errorf("Expected error string to include the word 'duplicate', but got '%s'", err.Error())
		}
	}
}
