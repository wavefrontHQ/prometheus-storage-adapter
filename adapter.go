package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/prometheus-storage-adapter/backend"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
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
			log.Fatal("tags must be formatted as \"tag=value,tag2=value...\"")
		}
		tags[strings.TrimSpace(l[0])] = strings.TrimSpace(l[1])
	}
	return tags
}

// struct to define the custom type to handle the arguments in key=value pair format.
type metricsFilter struct {
	filter map[string]string
}

// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (fl *metricsFilter) String() string {
	return fmt.Sprint(*fl)
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
func (fl *metricsFilter) Set(value string) error {
	//We are going to expect the arguments in <key>=<value> comma separated pairs.
	//We will then split them and assign it to staticMap for further filtering
	//in the backend.
	tmp := make(map[string]string, 0)
	for _, dt := range strings.Split(value, ",") {
		p := strings.Split(dt, "=")
		if len(p) < 2 {
			log.Fatal("arguments to override always needs to be in : 'key1=value1,key2=value' format")
		}
		tmp[p[0]] = p[1]
	}
	fl.filter = tmp
	return nil
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
	var convertPaths bool
	var convertTagPaths bool
	var metricsFilter metricsFilter

	flag.StringVar(&prefix, "prefix", "", "Prefix for metric names. If omitted, no prefix is added.")
	flag.StringVar(&proxy, "proxy", "", "Host address to Wavefront proxy.")
	flag.IntVar(&port, "proxy-port", 2878, "Proxy port.")
	flag.StringVar(&listen, "listen", "", "Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.")
	flag.StringVar(&tags, "tags", "", "A comma-separated list of tags to be added to each point on the form \"tag1=value1,tag2=value2...\".")
	flag.StringVar(&url, "url", "", "Wavefront URL for direct ingestion.")
	flag.StringVar(&token, "token", "", "Wavefront API token for direct ingestion.")
	debug := flag.Bool("debug", false, "Print detailed debug messages.")
	flag.IntVar(&batchSize, "batch-size", 0, "Metric sending batch size (ignored in proxy mode).")
	flag.IntVar(&bufferSize, "buffer-size", 0, "Metric buffer size (ignored in proxy mode).")
	flag.IntVar(&flushInterval, "flush-interval", 0, "Metric flush interval (in seconds).")
	flag.BoolVar(&convertPaths, "convert-paths", true, "Convert metric names to use period instead of underscores.")
	flag.BoolVar(&convertTagPaths, "convert-tag-paths", true, "Convert metric tags to use period instead of underscores.")
	flag.Var(&metricsFilter, "metrics-name-override", " list of name and overrides in the format 'key1=value1, key2,value2...'\n"+
		" key = original name of the metrics which is coming from prometheus \n"+
		" value =  name user wish to override with \n"+
		" no prefix and pathConversion will be applied to these metrics.")

	flag.Parse()

	if proxy == "" && url == "" {
		log.Fatal("Proxy address or Wavefront URL must be specified.")
	}

	if proxy != "" && url != "" {
		log.Fatal("Proxy address and Wavefront URL are mutually exclusive.")
	}

	if url != "" && token == "" {
		log.Fatal("API token must be specified for direct ingestion.")
	}

	if listen == "" {
		log.Fatal("Listening port must be specified using the -listen flag.")
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
	mw := backend.NewMetricWriter(sender, prefix, parseTags(tags), convertPaths, convertTagPaths, metricsFilter.filter)

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
