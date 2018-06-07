SHELL := /bin/bash

VERSION=$(shell grep -e 'Version' blog.go | head -n 1 | cut -d '"' -f 2)
BUILD=$(shell git describe --always)
CURDIR=$(shell pwd)

# Inject the build version (commit hash) into the executable.
LDFLAGS := -ldflags "-X main.Build=$(BUILD)"

# `make setup` to set up a new environment, pull dependencies, etc.
.PHONY: setup
setup: clean
	go get -u ./...

# `make build` to build the binary.
.PHONY: build
build:
	gofmt -w .
	go build $(LDFLAGS) -i -o bin/blog cmd/blog/main.go

# `make run` to run it in debug mode.
.PHONY: run
run:
	./go-reload cmd/blog/main.go -debug user-root

# `make test` to run unit tests.
.PHONY: test
test:
	go test ./...

# `make clean` cleans everything up.
.PHONY: clean
clean:
	rm -rf bin dist

# `make hardclean` cleans EVERY THING, including root/.private, resetting
# your database in the local dev environment. Be careful!
.PHONY: hardclean
hardclean: clean
	rm -rf root/.private

# `make docker.build` to build the Docker image
.PHONY: docker.build
docker.build:
	docker build -t blog .

# `make docker.run` to run the docker image
.PHONY: docker.run
docker.run:
	docker run --rm --name blog_debug -p 8000:80 -v $(CURDIR)/user-root:/data/www blog
