package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
)

type metricJSON struct {
	Name      string                 `json:"name"`
	Fields    map[string]interface{} `json:"fields"`
	Tags      map[string]string      `json:"tags"`
	Timestamp int64                  `json:"timestamp"`
}

type metricsStruct struct {
	Metrics []metricJSON `json:"metrics"`
}

// Convert takes the json output from telegraf and
// generates telegraf metrics from them that can be converted into any
// data format supported by telegraf.
func Convert(metricJSONBytes []byte) ([]telegraf.Metric, error) {
	bucket := metricsStruct{
		Metrics: make([]metricJSON, 0),
	}

	err := json.Unmarshal(metricJSONBytes, &bucket)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal Error: %s", err)
	}

	return makeMetricsList(bucket.Metrics)
}

func makeMetricsList(rawMetrics []metricJSON) ([]telegraf.Metric, error) {
	metricList := []telegraf.Metric{}

	for _, mtc := range rawMetrics {
		m, err := metric.New(
			mtc.Name,
			mtc.Tags,
			mtc.Fields,
			time.Unix(0, int64(time.Millisecond)*mtc.Timestamp))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create metric, skipping. Error was: %s", err)
		}
		metricList = append(metricList, m)
	}
	if len(metricList) == 0 {
		return nil, fmt.Errorf("No metrics have been successfully generated")
	}
	return metricList, nil
}
