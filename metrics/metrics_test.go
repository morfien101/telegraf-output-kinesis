package metrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestGenerateMetrics(t *testing.T) {
	out, err := ioutil.ReadFile("./testdata.json")
	if err != nil {
		t.Logf("%s", err)
		t.FailNow()
	}

	bucket := metricsStruct{
		Metrics: make([]metricJSON, 0),
	}

	err = json.Unmarshal(out, &bucket)
	if err != nil {
		t.Logf("%s", fmt.Errorf("Unmarshal Error: %s", err))
	}

	metricsList, err := Convert(out)

	t.Logf("%d metrics returned", len(metricsList))

	if len(metricsList) != len(bucket.Metrics) {
		t.Logf("Didn't get the right number of metrics")
		t.Fail()
	}
}

func BenchmarkMakeMetrics(b *testing.B) {
	out, err := ioutil.ReadFile("./testdata.json")
	if err != nil {
		b.Logf("%s", err)
		b.FailNow()
	}

	bucket := metricsStruct{
		Metrics: make([]metricJSON, 0),
	}

	err = json.Unmarshal(out, &bucket)
	if err != nil {
		b.Logf("%s", fmt.Errorf("Unmarshal Error: %s", err))
		b.FailNow()
	}

	for i := 0; i < b.N; i++ {
		_, err := makeMetricsList(bucket.Metrics)
		if err != nil {
			b.Logf("Failed to generate the metrics. Error: %s", err)
			b.FailNow()
		}
	}
}
