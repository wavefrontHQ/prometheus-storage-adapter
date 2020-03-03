package backend

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

type testCase struct {
	metric      string
	finalMetric string
	source      string
	finalSource string
	tags        []string
	finalTags   []string
}

var timestamp = int64(1146225600)

var testCases = []testCase{
	{
		metric:      "cpu_utilization_percent",
		finalMetric: "prom_cpu_utilization_percent",
		source:      "localhost",
		finalSource: "localhost",
		tags: []string{
			"bar=foo",
			"foo=bar",
			"cpu=1",
		},
		finalTags: []string{
			"\"bar\"=\"foo\"",
			"\"foo\"=\"bar\"",
			"\"cpu\"=\"1\"",
		},
	},
	{
		metric:      "cpu.utilization.gigatons",
		finalMetric: "prom_cpu.utilization.gigatons",
		source:      "this.is.a.host.com",
		finalSource: "this.is.a.host.com",
		tags: []string{
			"bar!foo=foo!bar",
			"foo_bar=bar_foo",
			"number.of.cpus=1",
		},
		finalTags: []string{
			"\"bar-foo\"=\"foo!bar\"",
			"\"foo_bar\"=\"bar_foo\"",
			"\"number.of.cpus\"=\"1\"",
		},
	},
	{
		metric:      "Heavily!@#$%Subbed",
		finalMetric: "prom_Heavily-----Subbed",
		source:      "some)(*&^source",
		finalSource: "some-----source",
		tags: []string{
			"bar!@#$%foo=foo!bar",
			"foo_bar=bar_foo",
			"number.of.cpus=1",
		},
		finalTags: []string{
			"\"bar-----foo\"=\"foo!bar\"",
			"\"foo_bar\"=\"bar_foo\"",
			"\"number.of.cpus\"=\"1\"",
		},
	},
}

func TestRoundtrips(t *testing.T) {
	var conn net.Conn
	response := make(chan (string))
	// Spin up a minimal listener to simulate a proxy
	go func() {
		ln, err := net.Listen("tcp", ":4711")
		require.NoError(t, err)
		for {
			conn, err = ln.Accept()
			if err != nil {
				// handle error
			}
			br := bufio.NewReader(conn)
			for {
				// Yes. Infinite loop.
				// We'll close the connection when we're done listening. This will drop us out of here.
				s, err := br.ReadString('\n')
				if err != nil {
					if strings.Contains(err.Error(), "use of closed network connection") {
						return
					}
					t.Error(err)
				}
				response <- s
			}
		}
	}()

	sender, err := senders.NewProxySender(
		&senders.ProxyConfiguration{
			Host:                 "localhost", MetricsPort:          4711,
		})
	require.NoError(t, err)
	w :=  NewMetricWriter(sender, "prom", map[string]string{})
	for _, test := range testCases {
		ts := prompb.TimeSeries{
			Labels: []prompb.Label{
				prompb.Label{
					Name:  "__name__",
					Value: test.metric,
				},
				prompb.Label{
					Name:  "instance",
					Value: test.source,
				},
			},
			Samples: []prompb.Sample{
				prompb.Sample{
					Value:     50,
					Timestamp: timestamp,
				},
			},
		}
		for _, tag := range test.tags {
			pair := strings.Split(tag, "=")
			ts.Labels = append(ts.Labels, prompb.Label{
				Name:  pair[0],
				Value: pair[1],
			})
		}
		req := prompb.WriteRequest{
			Timeseries: []prompb.TimeSeries{ts},
		}
		w.Write(req)
	}

	// Wait for replies
	for _, test := range testCases {
		select {
		case <-time.After(10 * time.Second):
			t.Error("Timed out waiting for metrics")
		case s := <-response:
			linePrefix := fmt.Sprintf("\"%s\" 50 %d source=\"%s\"", test.finalMetric, timestamp, test.finalSource)
			require.Equal(t, linePrefix, s[0:len(linePrefix)])
			r := []rune(s)
			require.Equal(t, '\n', r[len(r)-1])
			parts := append(strings.Split(linePrefix, " "), test.finalTags...)
			require.ElementsMatch(t, parts, strings.Split(strings.Trim(s, "\n"), " "))
		}
	}
	conn.Close()
}
