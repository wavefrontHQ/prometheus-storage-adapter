package backend

import (
	"bufio"
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

func TestWrite(t *testing.T) {
	m := metricPoint{
		Metric: "decontribulator.temperature",
		Source: "main decontribulator",
		Value:  42,
		Tags:   map[string]string{"carbendingulator": "S-1103347P1"},
	}
	w := MetricWriter{
		prefix: "xyz",
	}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	if err := w.writeMetricPoint(bw, &m); err != nil {
		t.Errorf("Error while writing metrics: %s", err)
	}
	bw.Flush()
	if buf.String() != "decontribulator.temperature 42.000000 0 source=\"main decontribulator\" carbendingulator=\"S-1103347P1\"\n" {
		t.Errorf("Resulting string was: %s", buf.String())
	}
}

func TestSanitizeName(t *testing.T) {
	s := "ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvxyz-"
	if ss := sanitizeName(s); ss != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = "this-IS-a-ReAlLy-COMPLEX-name-that-SHOULD-be-LEFT-unTOUCHED"
	if ss := sanitizeName(s); ss != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName(" this has a_leading*illegal(char")
	if "-this-has-a.leading-illegal-char" != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName("this has a_trailing*illegal(char=")
	if "this-has-a.trailing-illegal-char-" != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName("underscores_should_be_turned_into_periods")
	if "underscores.should.be.turned.into.periods" != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName("Some Chinese: 波前的岩石")
	if "Some-Chinese-------" != s {
		t.Errorf("Resulting string was: %s", s)
	}
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
	w := NewMetricWriter("localhost:4711", "prom", map[string]string{"foo": "bar", "bar": "foo"})
	ts := prompb.TimeSeries{
		Labels: []*prompb.Label{
			&prompb.Label{
				Name:  "__name__",
				Value: "cpu_utilization_percent",
			},
			&prompb.Label{
				Name:  "instance",
				Value: "localhost",
			},
			&prompb.Label{
				Name:  "cpu",
				Value: "1",
			},
		},
		Samples: []*prompb.Sample{
			&prompb.Sample{
				Value:     50,
				Timestamp: 1086062400,
			},
		},
	}
	req := prompb.WriteRequest{
		Timeseries: []*prompb.TimeSeries{&ts},
	}
	w.Write(req)

	// Wait for reply
	select {
	case <-time.After(10 * time.Second):
		t.Error("Times out waiting for metrics")
	case s := <-response:
		if s != "prom.cpu.utilization.percent 50.000000 1086062400 source=\"localhost\" foo=\"bar\" bar=\"foo\" cpu=\"1\"\n" {
			t.Errorf("Received from sender: %s", s)
		}
	}
}
