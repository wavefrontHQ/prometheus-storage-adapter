package backend

import (
	"bufio"
	"github.com/stretchr/testify/require"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

var linePrefix = "\"prom_cpu_utilization_percent\" 50 1086062400 source=\"localhost\""

var parts = []string {
	"\"prom_cpu_utilization_percent\"",
	"50",
	"1086062400",
	"source=\"localhost\"",
	"\"bar\"=\"foo\"",
	"\"foo\"=\"bar\"",
	"\"cpu\"=\"1\"",
}

func TestFull(t *testing.T) {
	response := make(chan (string))
	// Spin up a minimal listener to simulate a proxy
	go func() {
		ln, err := net.Listen("tcp", ":4711")
		if err != nil {
			// handle error
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				// handle error
			}
			defer conn.Close()
			br := bufio.NewReader(conn)
			s, err := br.ReadString('\n')
			if err != nil {
				t.Error(err)
			}
			response <- s
		}
	}()

	// Create a client and send a single metric through
	w, err := NewMetricWriter("localhost", 4711, "prom", map[string]string{"foo": "bar", "bar": "foo"})
	require.NoError(t, err)
	ts := prompb.TimeSeries{
		Labels: []prompb.Label{
			prompb.Label{
				Name:  "__name__",
				Value: "cpu_utilization_percent",
			},
			prompb.Label{
				Name:  "instance",
				Value: "localhost",
			},
			prompb.Label{
				Name:  "cpu",
				Value: "1",
			},
		},
		Samples: []prompb.Sample{
			prompb.Sample{
				Value:     50,
				Timestamp: 1086062400,
			},
		},
	}
	req := prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{ts},
	}
	w.Write(req)

	// Wait for reply
	select {
	case <-time.After(10 * time.Second):
		t.Error("Times out waiting for metrics")
	case s := <-response:
		require.Equal(t, linePrefix, s[0:len(linePrefix)])
		r := []rune(s)
		require.Equal(t, '\n', r[len(r)-1])
		require.ElementsMatch(t, parts, strings.Split(strings.Trim(s, "\n"), " "))
	}
}
