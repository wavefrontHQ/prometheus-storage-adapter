 # Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
DOCKER=docker
BINARY_NAME=adapter
TARGET=target
BINARY_LINUX=$(TARGET)/$(BINARY_NAME)_linux
BINARY_DARWIN=$(TARGET)/$(BINARY_NAME)_darwin
BINARY_WINDOWS=$(TARGET)/$(BINARY_NAME)_windows.EXE

DOCKER_REPO=wavefronthq
DOCKER_IMAGE=prometheus-storage-adapter
# Represents the upcoming version
# IMPORTANT: This is also overwritten by the release pipeline build with parameters
VERSION?=1.0.10

all: tidy build test

.PHONY build:
build: 
	$(GOBUILD) -o $(BINARY_NAME) -v

.PHONY test:
test: 
	$(GOTEST) -v ./...

.PHONY clean:
clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(TARGET)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

.PHONY fmt:
fmt:
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs gofmt -s -w

@PHONE tidy:
tidy:
	go mod tidy

build-all: tidy build-linux build-darwin build-windows
build-docker: build-linux
	$(DOCKER) build -t $(DOCKER_REPO)/$(DOCKER_IMAGE):$(VERSION) .
	$(DOCKER) tag $(DOCKER_REPO)/$(DOCKER_IMAGE):$(VERSION) $(DOCKER_REPO)/$(DOCKER_IMAGE):latest
release: build-all build-docker

# Cross compilation
.PHONY build-linux:
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) -v

.PHONY build-darwin:
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_DARWIN) -v

.PHONY build-windows:
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WINDOWS) -v
