package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/wavefrontHQ/prometheus-storage-adapter/backend"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/prometheus/prompb"
)

func main() {
	var prefix string
	var host string
	var listen string
	flag.StringVar(&prefix, "prefix", "", "Prefix for metric names. If omitted, no prefix is added.")
	flag.StringVar(&host, "proxy", "", "Address to wavefront proxy on the form 'hostname:port'.")
	flag.StringVar(&listen, "listen", "", "Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.")
	debug := flag.Bool("debug", false, "Print detailed debug messages.")
	//help := flag.Bool("help", false, "Print helpful information and exit.")
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
	mw := backend.NewMetricWriter(host, prefix)
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

		if err := mw.Write(req); err != nil {
			log.Errorf("Error during write to Wavefront:, %s", err)
		}
	})

	log.Fatal(http.ListenAndServe(listen, nil))
}
