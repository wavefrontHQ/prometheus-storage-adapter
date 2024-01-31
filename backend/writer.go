package backend

import (
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/prometheus/prompb"

	"github.com/wavefronthq/wavefront-sdk-go/senders"
)

type MetricWriter struct {
	prefix          string
	tags            map[string]string
	sender          senders.Sender
	convertPaths    bool
	convertTagPaths bool
	metricsSent     int64
	numErrors       int64
	errorRate       float64
	filters         map[string]string
}

var tagValueReplacer = strings.NewReplacer("\"", "\\\"", "*", "-")

func NewMetricWriter(sender senders.Sender, prefix string, tags map[string]string, convertPaths bool, convertTagPaths bool, filters map[string]string) *MetricWriter {
	return &MetricWriter{
		sender:          sender,
		prefix:          prefix,
		tags:            tags,
		convertPaths:    convertPaths,
		convertTagPaths: convertTagPaths,
		filters:         filters,
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
		tagName := w.buildTagName(l.Name)
		tags[tagName] = l.Value
	}
	fieldName := w.buildMetricName(tags["__name__"])
	delete(tags, "__name__")
	for _, value := range ts.Samples {
		// Prometheus sometimes sends NaN samples. We interpret them as
		// missing data and simply skip them.
		if math.IsNaN(value.Value) {
			continue
		}
		source, finalTags := w.buildTags(tags)
		err := w.sender.SendMetric(
			fieldName,
			value.Value,
			roundUpToNearestSecond(value.Timestamp),
			source,
			finalTags)
		if err != nil {
			log.Warnf("Cannot send metric: %s. Reason: %s. Skipping to next", fieldName, err)
		}
	}
}

func (w *MetricWriter) buildMetricName(name string) string {
	//if the metric is present in the filter then we are going to ignore it.
	// We are going to return the name which came in filter, a custom value.
	// No prefix should be appended to this metrics.
	//if user by mistake just pass "key1=" we are going to let it through normal process.

	if len(w.filters) != 0 {
		if val, ok := w.filters[name]; ok {
			if val != "" {
				return val
			} else {
				log.Debugf("filter %s came with out value, this is incorrect.", name)
			}
		}
	}

	if w.prefix != "" {
		name = w.prefix + "_" + name
	}
	if w.convertPaths {
		name = strings.Replace(name, "_", ".", -1)
	}
	return name
}

func (w *MetricWriter) buildTagName(name string) string {
	if name != "__name__" && w.convertTagPaths {
		name = strings.Replace(name, "_", ".", -1)
	}
	return name
}

func (w *MetricWriter) buildTags(mTags map[string]string) (string, map[string]string) {
	// Remove all empty tags.
	for k, v := range mTags {
		if v == "" {
			log.Debugf("dropping empty tag %s", k)
			delete(mTags, k)
		}
	}

	source := ""
	if val, ok := mTags["instance"]; ok {
		source = val
		delete(mTags, "instance")
	}

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
	err := w.sender.SendMetric("prom.gateway.healthcheck", 0, time.Now().Unix(), "", tags)
	if err != nil {
		return 503, err.Error()
	}
	return 200, "OK"
}

// roundUpToNearestSecond rounds milliseconds up to the nearest second and
// returns that value in millisecons. So 123000 -> 123000 and 123001 -> 124000
func roundUpToNearestSecond(milliseconds int64) int64 {
	return ((milliseconds + 1000 - 1) / 1000) * 1000
}
