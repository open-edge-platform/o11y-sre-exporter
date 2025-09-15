// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package scraping

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

const inputString = `# HELP orch_edgenode_mem_used_percent Percentage of used memory vs total memory on Edge Node host
# TYPE orch_edgenode_mem_used_percent gauge
orch_edgenode_mem_used_percent{customer="test-customer",host="9035e67dbe",hostGuid="",service="orch_edgenode"} 10
# HELP otelcol_exporter_sent_metric_points Number of metric points successfully sent to destination.
# TYPE otelcol_exporter_sent_metric_points counter
otelcol_exporter_sent_metric_points{customer="test-customer",host="9035e67dbe",hostGuid="",service="orch_edgenode"} 100

`

const delta = float64(0.0001)

func TestScrapeMetrics(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, inputString)
	}))
	var metric = models.ScrapeEventCondition{
		Metric:     "otelcol_exporter_sent_metric_points",
		CheckType:  models.ValueThreshold,
		Threshold:  0,
		MetricType: "counter",
	}
	t.Run("test ScrapeMetrics function", func(t *testing.T) {
		var scrapedMetricsList = []models.ScrapeEventCondition{metric}
		scrapingManager := NewScrapeEventDesc(svr.URL, scrapedMetricsList)
		err := scrapingManager.ScrapeMetrics()
		require.NoError(t, err)
	})

	t.Run("negative test ScrapeMetrics function - invalid endpoint", func(t *testing.T) {
		var scrapedMetricsList = []models.ScrapeEventCondition{metric}
		scrapingManager := NewScrapeEventDesc("", scrapedMetricsList)
		err := scrapingManager.ScrapeMetrics()
		require.Error(t, err)
	})
}

func TestGetMetrics(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, inputString)
	}))

	t.Run("test getMetrics function", func(t *testing.T) {
		output, err := getMetrics(svr.URL)
		require.NoError(t, err)
		require.Equal(t, inputString, output)
	})

	t.Run("negative test getMetrics function - endpoint with invalid chars", func(t *testing.T) {
		_, err := getMetrics("https://foobar.internal\n")
		require.Error(t, err)
	})

	t.Run("negative test getMetrics function - empty endpoint", func(t *testing.T) {
		_, err := getMetrics("")
		require.Error(t, err)
	})
}

func TestParseMetrics(t *testing.T) {
	t.Run("test parseMetrics function", func(t *testing.T) {
		parser := expfmt.NewTextParser(model.LegacyValidation)
		metricFamiliesByNameExpected, err := parser.TextToMetricFamilies(strings.NewReader(inputString))
		if err != nil {
			t.Fatal(err)
		}

		var metricFamiliesExpected []*dto.MetricFamily
		for _, mf := range metricFamiliesByNameExpected {
			metricFamiliesExpected = append(metricFamiliesExpected, mf)
		}

		metricFamiliesByNameActual, err := parseMetrics(inputString)
		require.NoError(t, err)

		var metricFamiliesActual []*dto.MetricFamily
		for _, mf := range metricFamiliesByNameActual {
			metricFamiliesActual = append(metricFamiliesActual, mf)
		}

		require.ElementsMatch(t, metricFamiliesExpected, metricFamiliesActual)
	})

	t.Run("negative test parseMetrics function", func(t *testing.T) {
		_, err := parseMetrics("foobar")
		require.Error(t, err)
	})
}

func TestGetValue(t *testing.T) {
	parser := expfmt.NewTextParser(model.LegacyValidation)
	metricFamiliesByName, err := parser.TextToMetricFamilies(strings.NewReader(inputString))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test getValue function - gauge", func(t *testing.T) {
		var metric = models.ScrapeEventCondition{
			Metric:     "orch_edgenode_mem_used_percent",
			CheckType:  models.ValueThreshold,
			Threshold:  6,
			MetricType: "gauge",
		}
		value, emptyMetrics, err := getValue(metric, metricFamiliesByName)

		require.NoError(t, err)
		require.InDelta(t, float64(10), value, delta)
		require.False(t, emptyMetrics)
	})

	t.Run("test getValue function - counter", func(t *testing.T) {
		var metric = models.ScrapeEventCondition{
			Metric:     "otelcol_exporter_sent_metric_points",
			CheckType:  models.ValueThreshold,
			Threshold:  6,
			MetricType: "counter",
		}
		value, emptyMetrics, err := getValue(metric, metricFamiliesByName)

		require.NoError(t, err)
		require.InDelta(t, float64(100), value, delta)
		require.False(t, emptyMetrics)
	})

	t.Run("test getValue function - metric does not exist", func(t *testing.T) {
		var metric = models.ScrapeEventCondition{
			Metric:     "foobar",
			CheckType:  models.ValueThreshold,
			Threshold:  6,
			MetricType: "gauge",
		}
		_, emptyMetric, err := getValue(metric, metricFamiliesByName)
		require.True(t, emptyMetric)
		require.NoError(t, err)
	})

	t.Run("test getValue function - unrecognized metricType", func(t *testing.T) {
		var metric = models.ScrapeEventCondition{
			Metric:     "orch_edgenode_mem_used_percent",
			CheckType:  models.ValueThreshold,
			Threshold:  6,
			MetricType: "foobar",
		}
		_, _, err := getValue(metric, metricFamiliesByName)
		require.Error(t, err)
	})
}

func TestEvalThreshold(t *testing.T) {
	var metricValue = models.ScrapeEventCondition{
		Metric:     "foo",
		CheckType:  models.ValueThreshold,
		Threshold:  0,
		MetricType: "bar",
	}
	var metricRate = models.ScrapeEventCondition{
		Metric:     "foo",
		CheckType:  models.RateThreshold,
		Threshold:  0,
		MetricType: "bar",
	}

	tests := map[string]struct {
		metric         models.ScrapeEventCondition
		inputValue     float64
		inputPrevValue float64
		inputPrevDiff  float64
		expectedString string
		expectedBool   bool
	}{
		"Test when threshold is exceeded for the first time - value threshold": {
			metric:         metricValue,
			inputValue:     5,
			inputPrevValue: 0,
			inputPrevDiff:  0,
			expectedString: fmt.Sprintf("Metric %q went above the threshold. Old value: %v New value: %v", metricValue.Metric, float64(0), float64(5)),
			expectedBool:   true,
		},
		"Test when threshold is exceeded not for first time - value threshold": {
			metric:         metricValue,
			inputValue:     5,
			inputPrevValue: 5,
			inputPrevDiff:  0,
			expectedString: "",
			expectedBool:   false,
		},
		"Test when threshold is not exceeded but previously was exceeded - value threshold": {
			metric:         metricValue,
			inputValue:     0,
			inputPrevValue: 5,
			inputPrevDiff:  0,
			expectedString: fmt.Sprintf("Metric %q went below the threshold. Old value: %v New value: %v", metricValue.Metric, float64(5), float64(0)),
			expectedBool:   true,
		},
		"Test when threshold is exceeded for the first time - rate threshold": {
			metric:         metricRate,
			inputValue:     5,
			inputPrevValue: 0,
			inputPrevDiff:  0,
			expectedString: fmt.Sprintf("Metric rate %q went above the threshold. Old rate: %v New rate: %v", metricRate.Metric, float64(0), float64(5)),
			expectedBool:   true,
		},
		"Test when threshold is exceeded not for first time - rate threshold": {
			metric:         metricRate,
			inputValue:     5,
			inputPrevValue: 0,
			inputPrevDiff:  5,
			expectedString: "",
			expectedBool:   false,
		},
		"Test when threshold is not exceeded but previously was exceeded - rate threshold": {
			metric:         metricRate,
			inputValue:     5,
			inputPrevValue: 5,
			inputPrevDiff:  5,
			expectedString: fmt.Sprintf("Metric rate %q went below the threshold. Old rate: %v New rate: %v", metricRate.Metric, float64(5), float64(0)),
			expectedBool:   true,
		},
		"Test when threshold is not exceeded and previously was not exceeded": {
			metric:         metricRate,
			inputValue:     0,
			inputPrevValue: 0,
			inputPrevDiff:  0,
			expectedString: "",
			expectedBool:   false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			returnBool, retrunString := evalThreshold(test.inputValue, test.inputPrevValue, test.inputPrevDiff, test.metric)
			require.Equal(t, test.expectedBool, returnBool)
			require.Equal(t, test.expectedString, retrunString)
		})
	}
}
