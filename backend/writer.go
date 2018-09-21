package backend

import (
	"bufio"
	"bytes"
	"math"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/prometheus/prompb"
)

type MetricWriter struct {
	prefix         string
	sourceOverride []string
	pool           *Pool
	tags           map[string]string
}

var tagValueReplacer = strings.NewReplacer("\"", "\\\"", "*", "-")

type metricPoint struct {
	Metric    string
	Value     float64
	Timestamp int64
	Source    string
	Tags      map[string]string
}

func NewMetricWriter(host, prefix string, tags map[string]string) *MetricWriter {
	return &MetricWriter{
		pool:   NewPool(host),
		prefix: prefix,
		tags:   tags,
	}
}

func (w *MetricWriter) Write(rq prompb.WriteRequest) error {
	out, err := w.pool.Get()
	if err != nil {
		return err
	}
	fail := false
	bw := bufio.NewWriter(out)
	defer func() {
		bw.Flush()
		if !fail {
			w.pool.Return(out)
		}
	}()
	for _, ts := range rq.Timeseries {
		if err := w.writeMetrics(bw, ts); err != nil {
			fail = true
			return err
		}
	}
	return nil
}

func (w *MetricWriter) writeMetrics(wrt *bufio.Writer, ts *prompb.TimeSeries) error {
	tags := make(map[string]string, len(ts.Labels))
	for _, l := range ts.Labels {
		tags[l.Name] = l.Value
	}
	fieldName := sanitizeName(tags["__name__"])
	delete(tags, "__name__")
	if w.prefix != "" {
		fieldName = w.prefix + "." + fieldName
	}
	fieldName = sanitizeName(fieldName)
	for _, value := range ts.Samples {
		// Prometheus sometimes sends NaN samples. We interpret them as
		// missing data and simply skip them.
		if math.IsNaN(value.Value) {
			continue
		}

		metric := &metricPoint{
			Metric:    fieldName,
			Timestamp: value.Timestamp,
		}

		metric.Value = value.Value

		source, tags := w.buildTags(tags)
		metric.Source = source
		metric.Tags = tags

		w.writeMetricPoint(wrt, metric)
	}
	return nil
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

func (w *MetricWriter) writeMetricPoint(wrt *bufio.Writer, metricPoint *metricPoint) error {
	buffer := bytes.NewBufferString("")
	buffer.WriteString(metricPoint.Metric)
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatFloat(metricPoint.Value, 'f', 6, 64))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatInt(metricPoint.Timestamp, 10))
	buffer.WriteString(" source=\"")
	buffer.WriteString(metricPoint.Source)
	buffer.WriteString("\"")

	for k, v := range metricPoint.Tags {
		buffer.WriteString(" ")
		buffer.WriteString(sanitizeName(k))
		buffer.WriteString("=\"")
		buffer.WriteString(tagValueReplacer.Replace(v))
		buffer.WriteString("\"")
	}
	log.Debugf(buffer.String())

	buffer.WriteString("\n")
	_, err := wrt.WriteString(buffer.String())
	return err
}

func sanitizeName(name string) string {
	var sb *bytes.Buffer
	for i, ch := range name {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' {
			if sb != nil {
				sb.WriteRune(ch)
			}
			continue
		}
		if sb == nil {
			sb = bytes.NewBufferString(name[:i])
		}
		if ch == '_' {
			sb.WriteRune('.')
		} else {
			sb.WriteRune('-')
		}
	}
	if sb == nil {
		return name
	}
	return sb.String()
}
