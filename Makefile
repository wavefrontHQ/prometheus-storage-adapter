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

all: build test
build: 
	$(GOBUILD) -o $(BINARY_NAME) -v
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(TARGET)
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
deps:
	$(GOGET) github.com/wavefrontHQ/prometheus-storage-adapter    
build-all: deps build-linux build-darwin build-windows
build-docker: build-linux
	$(DOCKER) build .
release: build-all build-docker

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) -v
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_DARWIN) -v
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WINDOWS) -v