# dev [![Go Report Card](https://goreportcard.com/badge/github.com/wish/dev)](https://goreportcard.com/report/github.com/wish/dev)

dev is a command line tool that provides a thin layer of porcelain on top of [Docker Compose](https://docs.docker.com/compose/).

# Background

# Requirements

 * Support sharing of docker-compose configuration across projects
 * Support sharing of networks across projects (i.e., manage creation of 'external' networks directly)
 * Support authentication with private container repositories
 * Support dependencies between projects, networks and registries

# Table of Contents
- [Installing](#installing)
- [Overview](#overview)
- [Commands](#commands)
  * [build](#build)
  * [up](#up)
  * [ps](#ps)
  * [sh](#sh)

# Installing

Binaries available for linux and and osx here.

# Contributing

You will need a current version of [golang](https://golang.org/dl/) that supports
modules to build this project.

# Configuration

Dev will search the current directory and its parent directory until it locates
a configuration file. The name of the configuration is .dev.yaml but can be
overridden with the --config flag. If a per-project configuration file cannot
be found, dev will look in your home directory and finally in
$XDG_CONFIG_HOME/dev for one.

If a configuration file is not found, dev will look in the current working
directory for a docker-compose.yml file. If one is found, it will create a dev
project with the base name of current directory.  For example, if you are
located in $HOME/Projects/my-app and there is a docker-compose.yml in that
directory, dev will create a command named 'my-app' and a number of
subcommands.  These subcommands can be listed by running `dev my-app --help`.

If you require more than one docker-compose.yml for your project, you can
specify these in the .dev.yaml file. For example, for the my-app project which
has a layout like this:

```
  $HOME/Projects/my-app:
    .dev.yaml
    docker-compose.yml

  $HOME/Projects/my-app/docker
    docker-compose.shared.yml
```

The $HOME/Projects/my-app/dev.yml might contain something like this:

 ```yaml
projects:
  my-app:
    docker_compose_files:
      - "docker/docker-compose.shared.yml"
      - "docker-compose.yml"
    depends_on: ["my-external-network"]

networks:
  my-external-network:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 173.16.242.0/16
 ```

Running 'dev my-app build' will provide both docker-compose.yml configuration
files to docker-compose with the -f flag.

When 'dev my-app up' is run, "my-external-network" will be created if it does not
exist.

Run 'dev my-app sh' to get a shell in the container or 'dev my-app sh ls -al'
to run 'ls -al' in the project container. The "project" container is the
container in the docker-compose.yml with the same name as the project in the
.dev.yaml file.

# Overview

Run `dev` to see a list of Projects


# Project Commands

The following commands are sub-commands added to each project added to
.dev.yaml. If no .dev.yaml could not be located, dev will look in the current
directory for a docker-compose.yml file and add a project with the same name as
the current directory.

## build

Run docker-compose build for the specified project. The build will specify
all the docker-compose files in the project's `docker_compose_files` array.

## ps

View details about the services running for the specified project. This is the
output of docker-compose ps for your project.

## up

Start the containers for the specified project. This will build or fetch the
images as required.

## sh

Run without arguments this command runs an interactive shell on the project
container. If run with arguments, the arguments are passed to the container shell
with the -c flag.

If this command is run from a subdirectory of the project directory this
command will first change directories such that relative commands from your
directory on the host can be run. If run from outside of your project
directory the starting directory defaults to the WORKDIR specified in the
project's Dockerfile.
