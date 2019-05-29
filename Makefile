DATE 		:= $(shell date +"%a %b %d %T %Y")
UNAME_S 	:= $(shell uname -s | tr A-Z a-z)
GOFILES_WATCH 	:= find . -type f -iname "*.go"
GOFILES_BUILD   := $(shell find . -type f -iname "*.go")
PKGS 		:= $(shell go list ./...)

VERSION := $(shell git describe --tags | cut -d'/' -f2 || echo "unreleased")
V_DIRTY := $(shell git describe --exact-match HEAD > /dev/null 2>&1 || echo "-unreleased")
SHA  	:= $(shell git rev-parse --short HEAD)


default: build/dev.${UNAME_S} build/dev ## Builds dev for your current operating system and runs tests

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: build/dev.linux build/dev.darwin ## Builds dev binaries for linux and osx

.PHONY: clean
clean: ## Removes all build artifacts
	rm -rf build ; go clean

.PHONY: lint
lint: ## Runs linter
	@golint $(PKGS)

.PHONY: vet
vet: ## Runs go vet
	@go vet $(PKGS)

.PHONY: test
test: lint vet ## Run static analysis and tests

build/dev.linux: $(**/*.go) test ## Creates the linux binary
	@GOOS=linux CGO_ENABLED=0 go build -ldflags \
	       '-X "github.com/wish/dev/cmd.BuildDate=${DATE}" -X "github.com/wish/dev/cmd.BuildSha=$(SHA)" -X "github.com/wish/dev/cmd.BuildVersion=$(VERSION)$(V_DIRTY)"' \
	       -o build/dev.linux cmd/dev/*

build/dev.darwin: $(**/*.go) test ## Creates the osx binary
	@GOOS=darwin CGO_ENABLED=0 go build -ldflags \
		'-X "github.com/wish/dev/cmd.BuildDate=${DATE}" -X "github.com/wish/dev/cmd.BuildSha=$(SHA)" -X "github.com/wish/dev/cmd.BuildVersion=$(VERSION)$(V_DIRTY)"' \
		-o build/dev.darwin cmd/dev/*

build/dev: ## Make a link to the executable for this OS type for convenience
	$(shell ln -s dev.${UNAME_S} build/dev)

.PHONY: watch
watch: ## Watch .go files for changes and rerun build (requires entr, see https://github.com/clibs/entr)
	$(GOFILES_WATCH) | entr -rc $(MAKE)
