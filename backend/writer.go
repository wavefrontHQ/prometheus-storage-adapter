package backend

import (
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"math"
	"strings"
	"time"

	"github.com/prometheus/prometheus/prompb"
	log "github.com/sirupsen/logrus"
)

type MetricWriter struct {
	prefix      string
	tags        map[string]string
	sender      senders.Sender
	metricsSent int64
	numErrors   int64
	errorRate   float64
}

var tagValueReplacer = strings.NewReplacer("\"", "\\\"", "*", "-")

func NewMetricWriter(sender senders.Sender, prefix string, tags map[string]string) *MetricWriter {
	return &MetricWriter{
		sender: sender,
		prefix: prefix,
		tags:   tags,
	}
}

func (w *MetricWriter) Write(rq prompb.WriteRequest) {
	for _, ts := range rq.Timeseries {
		w.writeMetrics(&ts)
	}
}

func (w *MetricWriter) writeMetrics(ts *prompb.TimeSeries) {
	tags := make(map[string]string, len(ts.Labels))
	for _, l := range ts.Labels {
		tags[l.Name] = l.Value
	}
	fieldName := tags["__name__"]
	delete(tags, "__name__")
	if w.prefix != "" {
		fieldName = w.prefix + "_" + fieldName
	}
	for _, value := range ts.Samples {
		// Prometheus sometimes sends NaN samples. We interpret them as
		// missing data and simply skip them.
		if math.IsNaN(value.Value) {
			continue
		}
		source, finalTags := w.buildTags(tags)
		err := w.sender.SendMetric(fieldName, value.Value, value.Timestamp, source, finalTags)
		if err != nil {
			log.Warnf("Cannot send metric: %s. Reason: %s. Skipping to next", fieldName, err)
		}
	}
}

func (w *MetricWriter) buildTags(mTags map[string]string) (string, map[string]string) {
	// Remove all empty tags.
	for k, v := range mTags {
		if v == "" {
			delete(mTags, k)
		}
	}
	source := mTags["instance"]
	delete(mTags, "instance")

	// Add optional tags
	for k, v := range w.tags {
		mTags[k] = v
	}

	return tagValueReplacer.Replace(source), mTags
}

func (w *MetricWriter) HealthCheck() (int, string) {
	tags := map[string]string{
		"test": "test",
	}
	err := w.sender.SendMetric("prom.gateway.healtcheck", 0, time.Now().Unix(), "", tags)
	if err != nil {
		return 503, err.Error()
	}
	return 200, "OK"
}
