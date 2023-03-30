# Prometheus Storage Adapter for Operations for Applications
[![build status][ci-img]][ci] [![Go Report Card][go-report-img]][go-report] [![Docker Pulls][docker-pull-img]][docker-img]

## Introduction
Prometheus Storage Adapters can act as a "fork" and send data to a secondary location. This adapter simply takes the data being sent to it and forwards it to a Wavefront proxy. It is useful when you want data collected by Prometheus to be available in Operations for Applications.

## Installation

### Helm Install for Kubernetes
Refer to the [Helm chart](https://github.com/wavefrontHQ/helm#installation) to install the Storage Adapter in Kubernetes.

### Download Binaries
Prebuilt binaries for Linux, macOS, and Windows are available [here](https://github.com/wavefrontHQ/prometheus-storage-adapter/releases).

### Build from Source
To build from source:

1. Download the source:
```
go get github.com/wavefronthq/prometheus-storage-adapter
```
2. Build it:
```
cd $(GOPATH)/src/github.com/wavefronthq/prometheus-storage-adapter
go mod tidy
go mod vendor
make build
```

## Configuration
The adapter takes the following parameters:
```
-batch-size int
    Metric sending batch size (ignored in proxy mode).
-buffer-size int
    Metric buffer size (ignored in proxy mode).
-convert-paths
    Convert metric names/tags to use period instead of underscores. (default true)
-debug
    Print detailed debug messages.
-flush-interval int
    Metric flush interval (in seconds).
-listen string
    Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.
-metrics-name-override value
     list of name and overrides in the format 'key1=value1, key2,value2...'
     key = original name of the metrics which is coming from prometheus 
     value =  name user wish to override with 
     no prefix and pathConversion will be applied to these metrics.
-prefix string
    Prefix for metric names. If omitted, no prefix is added.
-proxy string
    Host address to Wavefront proxy.
-proxy-port int
    Proxy port. (default 2878)
-tags string
    A comma-separated list of tags to be added to each point on the form "tag1=value1,tag2=value2...".
-token string
    Wavefront API token for direct ingestion.
-url string
    Wavefront URL for direct ingestion.
```

### Standalone Example
To have an adapter listen on port 1234 and forward the data to a Wavefront proxy running at localhost:2878 and use a metrics prefix of `prom`:
```
./adapter -proxy localhost -proxy-port 2878 -listen 1234 -prefix prom
```

## Docker Container Example
The adapter is available as a Docker image.

To run it as a Docker container with the parameters discussed above:
```
docker run wavefronthq/prometheus-storage-adapter -proxy=localhost -proxy-port=2878 -listen=1234 -prefix=prom -convert-paths=true
```

## Integrating with Prometheus
Integrating the adapter with Prometheus requires a small change to the `prometheus.yml` config file. All you have to do is to add these two lines to the end of `prometheus.yml`:

```
remote_write:
  - url: "http://localhost:1234/receive"
```
**Note:** Replace `localhost:1234` with the hostname/port of the Prometheus Storage Adapter.

Once you save the config file, restart Prometheus.


[ci-img]: https://github.com/wavefrontHQ/prometheus-storage-adapter/actions/workflows/go.yml/badge.svg
[ci]: https://github.com/wavefrontHQ/prometheus-storage-adapter/actions/workflows/go.yml
[go-report-img]: https://goreportcard.com/badge/github.com/wavefronthq/prometheus-storage-adapter
[go-report]: https://goreportcard.com/report/github.com/wavefronthq/prometheus-storage-adapter
[docker-pull-img]: https://img.shields.io/docker/pulls/wavefronthq/prometheus-storage-adapter.svg?logo=docker
[docker-img]: https://hub.docker.com/r/wavefronthq/prometheus-storage-adapter/
