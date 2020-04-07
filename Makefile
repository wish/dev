DATE 		:= $(shell date +"%a %b %d %T %Y")
UNAME_S 	:= $(shell uname -s | tr A-Z a-z)
GOFILES_WATCH 	:= find . -type f -iname "*.go"
GOFILES_BUILD   := $(shell find . -type f -iname "*.go" | grep -v '^.test\/')
PKGS 		:= $(shell go list -mod=vendor ./...)
PKGS_TO_TEST   	:= $(shell go list -mod=vendor ./... | grep -v test)


VERSION := $(shell git describe --tags 2> /dev/null || echo "unreleased")
V_DIRTY := $(shell git describe --exact-match HEAD 2> /dev/null > /dev/null || echo "-unreleased")
GIT     := $(shell git rev-parse --short HEAD)
DIRTY   := $(shell git diff-index --quiet HEAD 2> /dev/null > /dev/null || echo "-dirty")


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
	@golint -set_exit_status ${PKGS}

.PHONY: vet
vet: ## Runs go vet
	@go vet ${PKGS}

.PHONY: test
test:
	@go test -coverprofile coverage.txt -covermode=atomic ${PKGS_TO_TEST}

.PHONY: checks
checks: lint vet test ## Run static analysis and tests

build/dev.linux: $(GOFILES_BUILD) ## Creates the linux binary
	@GOOS=linux CGO_ENABLED=0 go build -ldflags \
	       '-X "github.com/wish/dev/cmd.BuildDate=${DATE}" -X "github.com/wish/dev/cmd.BuildSha=$(GIT)${DIRTY}" -X "github.com/wish/dev/cmd.BuildVersion=$(VERSION)${V_DIRTY}"' \
	       -o build/dev.linux -mod=vendor cmd/dev/*

build/dev.darwin: $(GOFILES_BUILD) ## Creates the osx binary
	@GOOS=darwin CGO_ENABLED=0 go build -ldflags \
		'-X "github.com/wish/dev/cmd.BuildDate=${DATE}" -X "github.com/wish/dev/cmd.BuildSha=$(GIT)${DIRTY}" -X "github.com/wish/dev/cmd.BuildVersion=$(VERSION)${V_DIRTY}"' \
		-o build/dev.darwin -mod=vendor cmd/dev/*

build/dev: ## Make a link to the executable for this OS type for convenience
	$(shell ln -s dev.${UNAME_S} build/dev)

.PHONY: watch
watch: ## Watch .go files for changes and rerun build (requires entr, see https://github.com/clibs/entr)
	$(GOFILES_WATCH) | entr -rc $(MAKE) checks ; $(MAKE)

.PHONY: coverage
coverage:
	@go test -coverprofile=/tmp/cover ${PKGS_TO_TEST}
	@go tool cover -html=/tmp/cover -o coverage.html
	@rm /tmp/cover
