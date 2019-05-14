# dev

dev is a command line tool that provides a thin layer of porcelain on top of [Docker Compose](https://docs.docker.com/compose/).

# Background

# Table of Contents
- [Installing](#installing)
- [Overview](#overview)
- [Commands](#commands)
  * [build](#build)
  * [up](#up)
  * [ps](#ps)
- [Troubleshooting](#troubleshooting)

# Installing

```bash
  go get github.com/wish/dev
```

# Configuration

Dev will search the current directory and its parent directory until it locates a configuration file. The name of the configuration
is .dev.toml but can be overridden with the --config flag. If a per-project configuration file cannot be found, dev will look in
your home directory and finally in $XDG_CONFIG_HOME/dev for one. If a configuration file is not found, dev will assume you want to use the
current directory as the only project directory and will search for docker-compose.yml files there. For each docker-compose.yml
found, dev will create a number of commands for the project. For example, if your docker-compose.yml is located in directory /home/shaw/Projects/foo
dev will create a command named 'foo' and a number of subcommands. These subcommands can be listed by running `dev foo --help`.

If you require more than one docker-compose.yml for your project, you can specify these in the .dev.toml file. For example,
   for the Foo project which has a layout like this:

```
   /home/shaw/Projects/foo:
      docker-compose.yml
  /home/shaw/Projects/foo/docker
      docker-compose.shared.yml
```

You might have /home/shaw/Projects/.dev.toml which contains something like this:

 ```toml

 [[projects]]
   name=foo
   docker_compose_files=[
      "docker/docker-compose.shared.yml",
      "docker-compose.yml",
     ]

 ```

Then when you run 'dev foo build', the resulting command would specify both docker-compose.yml configuration files to docker-compose



# Overview

Run `dev` to see a list of Projects

The commands are ordered by project.

# Project Commands

These commands are generated for each project that the dev tool locates. To run, you must first specify
the project name.

## build

Run docker-compose build for the specified project. This will the docker-compose.yml files found
for the project appended to the list of docker-compose.yml files specified in the configuration file
for this project, if one is specified.

## ps

View details about the services running for the specified project. This is the output of docker-compose ps
for your project.

## up

Start the containers for the specified project.
