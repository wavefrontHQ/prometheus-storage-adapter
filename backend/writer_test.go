package backend

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
)

type testCase struct {
	convertPaths bool
	metric       string
	finalMetric  string
	source       string
	finalSource  string
	filter       map[string]string
	tags         []string
	finalTags    []string
}

var timestamp = int64(1146225600)

var testCases = []testCase{
	{
		convertPaths: false,
		metric:       "cpu_utilization_percent",
		finalMetric:  "prom_cpu_utilization_percent",
		source:       "localhost",
		finalSource:  "localhost",
		filter:       make(map[string]string, 0),
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
		convertPaths: false,
		metric:       "cpu.utilization.gigatons",
		finalMetric:  "prom_cpu.utilization.gigatons",
		source:       "this.is.a.host.com",
		finalSource:  "this.is.a.host.com",
		filter:       make(map[string]string, 0),
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
		convertPaths: false,
		metric:       "Heavily!@#$%Subbed",
		finalMetric:  "prom_Heavily-----Subbed",
		source:       "some)(*&^source",
		finalSource:  "some)(-&^source",
		filter:       make(map[string]string, 0),
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
	{
		convertPaths: true,
		metric:       "status_request_per_second",
		finalMetric:  "status.request_per_second",
		source:       "some)(*&^source",
		finalSource:  "some)(-&^source",
		filter: map[string]string{
			"status_request_per_second": "status.request_per_second",
		},
		tags: []string{
			"bar!@#$%foo=foo!bar",
			"foo_bar=bar_foo",
			"number.of.cpus=1",
			"asset_id=123456789",
		},
		finalTags: []string{
			"\"bar-----foo\"=\"foo!bar\"",
			"\"foo_bar\"=\"bar_foo\"",
			"\"number.of.cpus\"=\"1\"",
			"\"asset_id\"=\"123456789\"",
		},
	},
	{
		convertPaths: false,
		metric:       "status.availability",
		finalMetric:  "prom_status.availability",
		source:       "some)(*&^source",
		finalSource:  "some)(-&^source",
		filter: map[string]string{
			"status_request_per_second": "status.request_per_second",
		},
		tags: []string{
			"bar!@#$%foo=foo!bar",
			"foo_bar=bar_foo",
			"number.of.cpus=1",
			"asset_id=123456789",
		},
		finalTags: []string{
			"\"bar-----foo\"=\"foo!bar\"",
			"\"foo_bar\"=\"bar_foo\"",
			"\"number.of.cpus\"=\"1\"",
			"\"asset_id\"=\"123456789\"",
		},
	},
}

func TestRoundtrips(t *testing.T) {
	var conn net.Conn
	response := make(chan string)
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
			Host: "localhost", MetricsPort: 4711,
		})
	require.NoError(t, err)
	for _, test := range testCases {
		w := NewMetricWriter(sender, "prom", map[string]string{}, test.convertPaths, test.filter)
		ts := prompb.TimeSeries{
			Labels: []prompb.Label{
				{
					Name:  "__name__",
					Value: test.metric,
				},
				{
					Name:  "instance",
					Value: test.source,
				},
			},
			Samples: []prompb.Sample{
				{
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
			linePrefix := fmt.Sprintf("\"%s\" 50 %d source=\"%s\"", test.finalMetric, roundUpToNearestSecond(timestamp), test.finalSource)
			require.Equal(t, linePrefix, s[0:len(linePrefix)])
			r := []rune(s)
			require.Equal(t, '\n', r[len(r)-1])
			parts := append(strings.Split(linePrefix, " "), test.finalTags...)
			require.ElementsMatch(t, parts, strings.Split(strings.Trim(s, "\n"), " "))
		}
	}
	conn.Close()
}

func TestBuildName(t *testing.T) {
	sender, err := senders.NewProxySender(
		&senders.ProxyConfiguration{
			Host: "localhost", MetricsPort: 4711,
		})
	require.NoError(t, err)

	testName := "metric_name_with_underscore"
	filters := make(map[string]string, 0)
	// empty prefix and convert=true
	w := NewMetricWriter(sender, "", map[string]string{}, true, filters)
	name := w.buildMetricName(testName)
	require.Equal(t, "metric.name.with.underscore", name)

	// empty prefix and convert=false
	w = NewMetricWriter(sender, "", map[string]string{}, false, filters)
	name = w.buildMetricName(testName)
	require.Equal(t, testName, name)

	// non-empty prefix and convert=true
	w = NewMetricWriter(sender, "prom", map[string]string{}, true, filters)
	name = w.buildMetricName(testName)
	require.Equal(t, "prom.metric.name.with.underscore", name)

	// non-empty prefix and convert=false
	w = NewMetricWriter(sender, "prom", map[string]string{}, false, filters)
	name = w.buildMetricName(testName)
	require.Equal(t, "prom_"+testName, name)

	//filter metrics that should  have  prefix
	w = NewMetricWriter(sender, "prom", map[string]string{}, true, filters)
	name = w.buildMetricName("metrics_availability_1")
	require.Equal(t, "prom.metrics.availability.1", name)

}
