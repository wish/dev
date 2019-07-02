package test

import (
	"fmt"

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

// CreateConfigFile creates the specified filename in the current working
// directory and returns a function that will remove it.
func CreateConfigFile(fs afero.Fs, content string, filename string) {
	err := afero.WriteFile(fs, filename, []byte(content), 0)
	if err != nil {
		fmt.Println(err)
	}
}
