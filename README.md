# Prometheus Storage Adapter for Wavefront [![build status][ci-img]][ci] [![Go Report Card][go-report-img]][go-report] [![Docker Pulls][docker-pull-img]][docker-img]

## Introduction
Prometheus storage adapters can act as a "fork" and send data to a secondary location. This adapter simply takes the data being sent to it and forwards it to a Wavefront proxy. It is useful when you want data collected by Prometheus to be available in Wavefront.

## Installation

### Helm install for Kubernetes
Refer to the [helm chart](https://github.com/wavefrontHQ/helm#installation) to install the storage adapter in Kubernetes.

### Download binaries
Prebuilt binaries for Linux, macOS and Windows are available [here](https://github.com/wavefrontHQ/prometheus-storage-adapter/releases).

### Building from source
To build from source:

1. Download the source:
```
go get github.com/wavefronthq/prometheus-storage-adapter
```
2. Build it:
```
cd $(GOPATH)/src/github.com/wavefronthq/prometheus-storage-adapter
make deps build
```

## Configuration
The adapter takes the following parameters:
```
-debug
    Print detailed debug messages.
-convert-paths
    Convert metric names/tags to use period instead of underscores. (default true)    
-listen string
    Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.
-prefix string
    Prefix for metric names. If omitted, no prefix is added.
-proxy string
    Host address to wavefront proxy.
-proxy-port int
    Proxy port. (default 2878)
-tags string
    A comma separated list of tags to be added to each point on the form "tag1=value1,tag2=value2..."
```

### Standalone example
To have an adapter listen on port 1234 and forward the data to a Proxy running at localhost:2878 and use a metrics prefix of `prom`:
```
./adapter -proxy localhost -proxy-port 2878 -listen 1234 -prefix prom
```

## Docker container example
The adapter is available as a Docker image.

To run it as a docker container with the parameters discussed above:
```
docker run wavefronthq/prometheus-storage-adapter -proxy localhost -proxy-port 2878 -listen 1234 -prefix prom
```

## Integrating with Prometheus
Integrating the adapter with Prometheus requires a small change to the prometheus.yml config file. All you have to do is to add these two lines to the end of prometheus.yml:

```
remote_write:
  - url: "http://localhost:1234/receive"
```
**Note:** Replace `localhost:1234` with the hostname/port of the prometheus storage adapter.

Restart Prometheus once you have saved the config file.

[ci-img]: https://travis-ci.com/wavefrontHQ/prometheus-storage-adapter.svg?branch=master
[ci]: https://travis-ci.com/wavefrontHQ/prometheus-storage-adapter
[go-report-img]: https://goreportcard.com/badge/github.com/wavefronthq/prometheus-storage-adapter
[go-report]: https://goreportcard.com/report/github.com/wavefronthq/prometheus-storage-adapter
[docker-pull-img]: https://img.shields.io/docker/pulls/wavefronthq/prometheus-storage-adapter.svg?logo=docker
[docker-img]: https://hub.docker.com/r/wavefronthq/prometheus-storage-adapter/
