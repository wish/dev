# dev  ![codecov][1] [![Build Status](https://travis-ci.org/wish/dev.svg?branch=master)](https://travis-ci.org/wish/dev) [![Go Report Card](https://goreportcard.com/badge/github.com/wish/dev)](https://goreportcard.com/report/github.com/wish/dev)

dev is a command line tool that wraps [Docker Compose](https://docs.docker.com/compose/) to enable shared resources
for an ideal development environment.


# Table of Contents
- [Background](#background)
- [Goals](#goals)
- [Installing](#installing)
  * [Ubuntu](#ubuntu)
  * [OSX](#osx)
- [Building](#Building)
- [Configuration](#Configuration)
  * [.dev.yml](#.dev.yaml)
- [{Project} Commands](#project-commands)
  * [build](#build)
  * [up](#up)
  * [ps](#ps)
  * [sh](#sh)
- [Contributing](#contributing)
- [License](#license)


# Background

Versions up to 2.1 of the docker-compose configuration file had a convenient
way to share configuration across docker-compose files using the
[extends](https://docs.docker.com/compose/extends/#extending-services) keyword.
For those wishing to adopt later versions of the configuration, the loss of the
extends keyword has been [problematic](https://github.com/moby/moby/issues/31101).
For some, the ability to specify multiple configurations with '-f' flag to
docker-compose is workable. Starting with version 3.4 another option is to
use [extension fields](https://docs.docker.com/compose/compose-file/#extension-fields).

If you find the above extensions insufficient for your development container
needs, you might find this project interesting.


# Goals

 * Support sharing of docker-compose configuration across projects
 * Support sharing of networks across projects (i.e., manage creation of 'external' networks directly)
 * Support authentication with private container repositories
 * Support dependencies between projects, networks and registries


# Installing

Binaries available for linux and and osx on the [releases](https://github.com/wish/dev/releases) page.

## Ubuntu

Dev is bundled as a deb and made available as a ppa on launchpad.


```bash
sudo add-apt-repository ppa:wishlaunchpad/ppa
sudo apt install dev
```

## OSX

Dev can be installed with Homebrew.

```bash
brew tap wish/homebrew-wish
brew install wish-dev
```

# Building

You will need a current version of [golang](https://golang.org/dl/) that supports
modules to build this project.

1. Clone this repository.
1. If you clone into your $GOPATH, you will need to enable go modules via the
[GO11MODULE](https://github.com/golang/go/wiki/Modules) environment variable.
1. Run `make help` for a list of targets.


# Configuration

The `dev` command will search the current directory and its parent directory
until it locates a configuration file. If no configuration file is found in
your home directory it will look in the `$XDG_CONFIG_HOME/dev` directory for
one. Valid dev configuration file names will match the following regular
expression: `.?dev.ya?ml`.

The search for a configuration file can also be overridden by specifying a path
via the DEV_CONFIG environment variable. This is the same mechanism you must use
if you would like to use more than one dev configuration file. To use more
than one configuration file, separate the .dev.yaml paths with a colon, i.e.:

```
export DEV_CONFIG=$HOME/Projects/app_one:$HOME/Projects/shared_app_config
```

### .dev.yaml

There are many ways to structure you project with the `dev` tool.

Typically you will have one .dev.yaml file for each project. Each .dev.yaml
can manage multiple projects if you happen to have multiple projects in
the same repository.

If you require more than one docker-compose.yml for your project you can
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
    depends_on: ["my-registry", "my-external-network"]

networks:
  my-external-network:
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 173.16.242.0/16

registries:
  my-registry:
      url: "https://my-registry.personal.com"
      username: "name"
      password: "pa$$word"
      continue_on_failure: True
 ```

Running 'dev my-app build' will attempt to login to `my-registry` before
running docker-compose build.

When `dev my-app up` is run `dev` will first create `my-external-network` if it
does not exist already, taking care to remove any existing containers listed in
the `docker_compose_files` that are connected to a network of the same name but
a different network id.

'dev my-app sh' will shell into the project container or run any commands specified on
container. 'dev my-app sh ls -al' will list all of the files in the project container.
The "project" container is the container in the docker-compose.yml with the
same name as the project in the .dev.yaml file.


# Project Commands

The following commands are added as sub-commands for each project defined in your
.dev.yaml file/s.

## build

Run docker-compose build for the specified project. The build will specify
all the docker-compose files in the project's `docker_compose_files` array
to the `docker-compose` command with the -f flag.

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
project's Dockerfile. This functionality currently assumes that mapped the
project directory (the one where the .dev.yaml file resides) into the
project container.


# Contributing

1. Fork it
1. Download your fork to your PC (`git clone https://github.com/your_username/dev && cd dev`)
1. Create your feature branch (`git checkout -b my-new-feature`)
1. Make changes and add them (`git add .`)
1. Commit your changes (`git commit -m 'Add some feature'`)
1. Push to the branch (`git push origin my-new-feature`)
1. Create new pull request

# License

Dev is released under the MIT license. See [LICENSE](https://github.com/wish/dev/blob/master/LICENSE)



[1]: https://codecov.io/gh/wish/dev/branch/master/graph/badge.svg
