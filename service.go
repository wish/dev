package dev

import (
	log "github.com/sirupsen/logrus"
	"github.com/wish/dev/compose"
	"github.com/wish/dev/config"
)

// CreateBuildableServiceList creates a list of buildable services in the projects docker-compose files.
func CreateBuildableServiceList(devConfig *config.Dev, project *config.Project) []string {

	serviceList := []string{}
	for _, composeFilename := range project.DockerComposeFilenames {
		composeConfig, err := compose.Parse(devConfig.GetFs(), project.Directory, composeFilename)
		if err != nil {
			log.Fatal("Failed to parse docker-compose appConfig file: ", err)
		}

		for _, service := range composeConfig.Services {
			if service.Build.Context != "" {
				serviceList = append(serviceList, service.Name)
			}
		}
	}
	return serviceList
}
