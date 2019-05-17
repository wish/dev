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
a configuration file. The name of the configuration is .dev.yml but can be
overridden with the --config flag. If a per-project configuration file cannot
be found, dev will look in your home directory and finally in
$XDG_CONFIG_HOME/dev for one. If a configuration file is not found, dev will
assume you want to use the current directory as the only project directory and
will search for docker-compose.yml files there. For each docker-compose.yml
found, dev will create a number of commands for the project. For example, if
your docker-compose.yml is located in directory /home/shaw/Projects/foo dev
will create a command named 'foo' and a number of subcommands. These
subcommands can be listed by running `dev foo --help`.

If you require more than one docker-compose.yml for your project, you can
specify these in the .dev.yml file. For example, for the Foo project which has
a layout like this:

```
  $HOME/Projects/foo:
    .dev.yaml
    docker-compose.yml

  $HOME/Projects/foo/docker
    docker-compose.shared.yml
```

The $HOME/Projects/foo/.dev.yml might contain something like this:

 ```yaml
projects:
  foo:
    docker_compose_files:
      - "docker/docker-compose.shared.yml"
      - "docker-compose.yml"
    depends: ["my-external-network"]

networks:
  my-external-network:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 173.16.242.0/16
 ```

Running 'dev foo build' will provide both docker-compose.yml configuration
files to docker-compose with the -f flag.

When 'dev foo up' is run, "my-external-network" will be created if it does not
exist.

Run 'dev foo sh' to get a shell in the container or 'dev foo sh <command>' to
run command in your container (assuming you've mapped your project directory as
a volume into the container.


# Overview

Run `dev` to see a list of Projects

The commands are ordered by project.

# Project Commands

These commands are generated for each project that the dev tool locates. To
run, you must first specify the project name.

## build

Run docker-compose build for the specified project. This will the
docker-compose.yml files found for the project appended to the list of
docker-compose.yml files specified in the configuration file for this project,
if one is specified.

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
