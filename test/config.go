package test

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/afero"
)

// BigCoConfig is a test dev configuration.
const BigCoConfig = `
log:
  level: "info"

image_prefix: "bigco"

projects:
  postgresql:
    aliases: ["db", "pg"]
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

// CreateFile creates the file specified by filename with the contents
// provided.
func CreateFile(fs afero.Fs, content string, filename string, perm os.FileMode) {
	err := afero.WriteFile(fs, filename, []byte(content), perm)
	if err != nil {
		fmt.Println(err)
	}
}

// CreateConfigFile creates the specified filename in the current working
// directory.
func CreateConfigFile(fs afero.Fs, content string, filename string) {
	CreateFile(fs, content, filename, 0)
}

// CreateDockerCompose creates a docker-compose file in the specified
// directory.
func CreateDockerCompose(fs afero.Fs, directory string) {
	CreateFile(fs, "", path.Join(directory, "docker-compose"), 0111)
}
