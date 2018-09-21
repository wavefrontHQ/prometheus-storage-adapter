package backend

import (
	"bytes"
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
	for _, ts := range rq.Timeseries {
		for _, metricPoint := range w.buildMetrics(ts) {
			metricLine := w.formatMetricPoint(metricPoint)
			_, err := out.Write([]byte(metricLine))
			if err != nil {
				// Writing to a pipe shouldn't give us an error, so we panic if we get one!
				panic(err)
			}
		}
	}
	w.pool.Return(out)
	return nil
}

func (w *MetricWriter) buildMetrics(ts *prompb.TimeSeries) []*metricPoint {
	ret := []*metricPoint{}

	tags := make(map[string]string, len(ts.Labels))
	for _, l := range ts.Labels {
		tags[l.Name] = l.Value
	}
	fieldName := strings.Replace(tags["__name__"], "_", ".", -1)
	delete(tags, "__name__")
	if w.prefix != "" {
		fieldName = w.prefix + "." + fieldName
	}
	fieldName = sanitizeName(fieldName)
	for _, value := range ts.Samples {
		metric := &metricPoint{
			Metric:    fieldName,
			Timestamp: value.Timestamp,
		}

		metric.Value = value.Value

		source, tags := w.buildTags(tags)
		metric.Source = source
		metric.Tags = tags

		ret = append(ret, metric)
	}
	return ret
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

func (w *MetricWriter) formatMetricPoint(metricPoint *metricPoint) string {
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

	buffer.WriteString("\n")

	log.Debugf(buffer.String())
	return buffer.String()
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
		sb.WriteRune('-')
	}
	if sb == nil {
		return name
	}
	return sb.String()
}
