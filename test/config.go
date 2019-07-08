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

// SharedCompose is a typical shared configuration that may be used by other
// apps in the same network.
const SharedCompose = `
version: '3.6'

# This configuration relies on the network configuration below:
#
# networks:
#     app-net:
#       external: true
#
# Network is created by the dev tool. See the project .dev.yaml.
#
services:
  jaeger:
    container_name: jaeger
    image: jaegertracing/all-in-one:1.8
    ports:
      - "6831:6831/udp"
      - "16686:16686"
    networks:
      app-net:
        ipv4_address: 173.16.242.50

  mongodb:
    container_name: mongodb
    image: mongo:3.4
    networks:
      app-net:
        ipv4_address: 173.16.242.12
`

// AppCompose is a typical compose config that may be used alone or
// in conjunction with shared configuration.
const AppCompose = `
version: '3.6'

networks:
  app-net:
    external: true

services:
  app:
    container_name: app
    build:
        context: .
        dockerfile: app.Dockerfile
    volumes:
      - .:/home/app:delegated
    environment:
      - "DOCKER_USER=${USER:-app}"

    networks:
      app-net:
        ipv4_address: 173.16.242.1
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

// CreateDockerComposeBinary creates a fake docker-compose file in the
// specified directory.
func CreateDockerComposeBinary(fs afero.Fs, directory string) {
	CreateFile(fs, "", path.Join(directory, "docker-compose"), 0111)
}
