SHA  := $(shell git rev-parse --short HEAD)
DATE := $(shell date +"%a %b %d %T %Y")
UNAME_S := $(shell uname -s | tr A-Z a-z)
GOFILES_WATCH := find . -type f -iname "*.go"
GOFILES_BUILD := $(shell find . -type f -iname "*.go")
GO_PACKAGES := $(shell go list ./...)

default: \
	build/${UNAME_S}/dev \
	build/dev

.PHONY: all
all:
	build/dev.linux
	build/dev.darwin

.PHONY: clean
clean:
	rm -rf build

lint:
	@golint $(GO_PACKAGES)

vet:
	@go vet $(GO_PACKAGES)


test: lint vet

build/linux/dev: ${GOFILES_BUILD} test
	@GOOS=linux CGO_ENABLED=0 go build -ldflags \
	       '-X "github.com/wish/dev/cmd.BuildDate=${DATE}" -X "github.com/wish/dev/cmd.BuildSha=${SHA}"' \
	       -o build/linux/dev cmd/dev/*

build/darwin/dev: ${GOFILES_BUILD} test
	@GOOS=darwin CGO_ENABLED=0 go build -ldflags \
		'-X "github.com/wish/dev/cmd.BuildDate=${DATE}" -X "github.com/wish/dev/cmd.BuildSha=${SHA}"' \
		-o build/darwin/dev cmd/dev/*

# make a link to the executable for this OS type for convience
build/dev:
	$(shell ln -s ${UNAME_S}/dev build/dev)

# Rerun build on change
.PHONY: watch
watch:
	$(GOFILES_WATCH) | entr -rc $(MAKE)
