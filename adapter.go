package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	var host string
	var listen string
	var tags string
	var port int
	flag.StringVar(&prefix, "prefix", "", "Prefix for metric names. If omitted, no prefix is added.")
	flag.StringVar(&host, "proxy", "", "Host address to wavefront proxy.")
	flag.IntVar(&port, "proxy-port", 2878, "Proxy port.")
	flag.StringVar(&listen, "listen", "", "Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.")
	flag.StringVar(&tags, "tags", "", "A comma separated list of tags to be added to each point on the form \"tag1=value1,tag2=value2...\"")
	debug := flag.Bool("debug", false, "Print detailed debug messages.")
	flag.Parse()

	if host == "" {
		fmt.Fprintln(os.Stderr, "Proxy address must be specified using the -proxy flag.")
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
	mw, err := backend.NewMetricWriter(host, port, prefix, parseTags(tags))
	if err != nil {
		log.Fatal(err)
		return
	}

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
