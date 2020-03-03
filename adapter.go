package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/wavefronthq/prometheus-storage-adapter/backend"

	"github.com/prometheus/prometheus/prompb"
	log "github.com/sirupsen/logrus"
)

type healthResult struct {
	Message string
}

func parseTags(s string) map[string]string {
	tags := make(map[string]string)
	if s == "" {
		return tags
	}
	for _, tag := range strings.Split(s, ",") {
		l := strings.Split(tag, "=")
		if len(l) != 2 {
			fmt.Fprintln(os.Stderr, "Tags must be formatted as \"tag=value,tag2=value...\"")
			os.Exit(1)
		}
		tags[strings.TrimSpace(l[0])] = strings.TrimSpace(l[1])
	}
	return tags
}

func main() {
	var prefix string
	var proxy string
	var listen string
	var tags string
	var port int
	var url string
	var token string
	var batchSize int
	var bufferSize int
	var flushInterval int
	flag.StringVar(&prefix, "prefix", "", "Prefix for metric names. If omitted, no prefix is added.")
	flag.StringVar(&proxy, "proxy", "", "Host address to wavefront proxy.")
	flag.IntVar(&port, "proxy-port", 2878, "Proxy port.")
	flag.StringVar(&listen, "listen", "", "Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.")
	flag.StringVar(&tags, "tags", "", "A comma separated list of tags to be added to each point on the form \"tag1=value1,tag2=value2...\"")
	flag.StringVar(&url, "url", "", "Wavefront URL for direct ingestion")
	flag.StringVar(&token, "token", "", "Wavefront API token for direct ingestion")
	debug := flag.Bool("debug", false, "Print detailed debug messages.")
	flag.IntVar(&batchSize, "batch-size", 0, "Metric sending batch size (ignored in proxy mode)")
	flag.IntVar(&bufferSize, "buffer-size", 0, "Metric buffer size (ignored in proxy mode")
	flag.IntVar(&flushInterval, "flush-interval", 0, "Metric flush interval (in seconds)")
	flag.Parse()

	if proxy == "" && url == "" {
		fmt.Fprintln(os.Stderr, "Proxy address or Wavefront URL must be specified.")
		os.Exit(1)
	}

	if proxy != "" && url != "" {
		fmt.Fprintln(os.Stderr, "Proxy address and Wavefront are mutually exclusive.")
		os.Exit(1)
	}

	if url != "" && token == "" {
		fmt.Fprintln(os.Stderr, "API token must be specified for direct ingestion.")
		os.Exit(1)
	}

	if listen == "" {
		fmt.Fprintln(os.Stderr, "Listening port must be specified using the -listen flag.")
		os.Exit(1)
	}
	if !strings.Contains(listen, ":") {
		listen = ":" + listen
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	var sender senders.Sender
	var err error
	if proxy != "" {
		sender, err = senders.NewProxySender(
			&senders.ProxyConfiguration{
				Host:                 proxy,
				MetricsPort:          port,
				FlushIntervalSeconds: flushInterval,
			})
	} else {
		sender, err = senders.NewDirectSender(
			&senders.DirectConfiguration{
				Server:               url,
				Token:                token,
				BatchSize:            batchSize,
				MaxBufferSize:        bufferSize,
				FlushIntervalSeconds: flushInterval,
			})
	}
	if err != nil {
		log.Fatal(err)
		return
	}
	mw := backend.NewMetricWriter(sender, prefix, parseTags(tags))

	// Install metric receiver
	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Debugf("Got request")

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mw.Write(req)
	})

	// Install health checker
	http.HandleFunc("/health", func(w http.ResponseWriter, request *http.Request) {
		status, message := mw.HealthCheck()
		result := healthResult{
			Message: message,
		}
		b, err := json.Marshal(&result)
		if err != nil {
			http.Error(w, "Irrecoverable error: "+err.Error(), 500)
			return
		}
		if status != 200 {
			http.Error(w, string(b), status)
		} else {
			fmt.Fprint(w, string(b))
		}
	})
	log.Fatal(http.ListenAndServe(listen, nil))
}
