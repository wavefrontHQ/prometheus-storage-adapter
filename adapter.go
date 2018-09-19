package main

import (
	"io/ioutil"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/wavefrontHQ/prometheus-storage-adapter/backend"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/prometheus/prompb"
)

func main() {
	log.SetLevel(log.DebugLevel)
	mw := backend.NewMetricWriter("localhost:2878")
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

	log.Fatal(http.ListenAndServe(":1234", nil))
}
