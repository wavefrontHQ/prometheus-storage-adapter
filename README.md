# Prometheus Storage Adapter for Wavefront

## Introduction
Prometheus storage adapters can act as a "fork" and send data to a secondary location. This adapter simply takes the data being sent to it and forwards it to a Wavefront proxy. It is useful when you want data collected by Prometheus to be available in Wavefront.

## Installation

### Download binaries
Prebuilt binaries for Linux, MacOSX and Windows are available here TODO: Link to releases

### Building from source
Building from source is easy. Simply grab the code with go get and build it with make.

1. Download the source
```
go get github.com/wavefrontHQ/prometheus-storage-adapter
```
2. Build it
```
cd $(GOPATH)/src/github.com/wavefrontHQ/prometheus-storage-adapter
make deps build
```

## Running the adapter
You can run the adapter directly from the command line, but in production you would probably make it a service that starts at system boot time. TODO: Publish start files.

The adapter takes the following parameters:
```
 -debug
    	Print detailed debug messages.
  -listen string
    	Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.
  -prefix string
    	Prefix for metric names. If omitted, no prefix is added.
  -proxy string
    	Host address to wavefront proxy.
  -proxy-port int
    	Proxy port. (default 2878)
```

### Example
To run the adapter listening to port 1234 and sending results to localhost:2878, we can use the following command. This command also adds a prefix ("prom") to all metrics coming from the adapter.
```
./adapter -proxy localhost -proxy-port 2878 -listen 1234 -prefix prom
```

## Integrating with Prometheus
Integrating the adapter with Prometheus only takes a small change to the prometheus.yml config file. All you have to do is to add these two lines to the end of prometheus.yml:

```
remote_write:
  - url: "http://localhost:1234/receive"
```

Once you have saved the config file, you need to restart Prometheus.
