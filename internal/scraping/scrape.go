// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package scraping

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

type ScrapeEventDesc struct {
	scrapeMetricsList []models.ScrapeEventCondition
	metricsEndpoint   string
	previousValues    []float64
	previousDiffs     []float64
}

func NewScrapeEventDesc(endpoint string, metricsList []models.ScrapeEventCondition) *ScrapeEventDesc {
	log.Printf("New ScrapeEventDesc registered for %v endpoint", endpoint)
	for i := range metricsList {
		log.Print(metricsList[i].ToString())
	}
	return &ScrapeEventDesc{
		scrapeMetricsList: metricsList,
		metricsEndpoint:   endpoint,
		previousValues:    make([]float64, len(metricsList)),
		previousDiffs:     make([]float64, len(metricsList)),
	}
}

func (sm *ScrapeEventDesc) ScrapeMetrics() error {
	bodyStr, err := getMetrics(sm.metricsEndpoint)
	if err != nil {
		return err
	}

	metricFamiliesByName, err := parseMetrics(bodyStr)
	if err != nil {
		return err
	}

	sm.logWarnings(metricFamiliesByName)

	return nil
}

func getMetrics(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return string(body), nil
}

func parseMetrics(text string) (map[string]*dto.MetricFamily, error) {
	parser := expfmt.NewTextParser(model.LegacyValidation)
	metricFamiliesByName, err := parser.TextToMetricFamilies(strings.NewReader(text))
	if err != nil {
		return nil, err
	}
	return metricFamiliesByName, nil
}

func (sm *ScrapeEventDesc) logWarnings(metricFamiliesByName map[string]*dto.MetricFamily) {
	for i := range sm.scrapeMetricsList {
		scrapeMetric := sm.scrapeMetricsList[i]
		previousDiff := sm.previousDiffs[i]
		previousValue := sm.previousValues[i]

		value, emptyMetric, err := getValue(scrapeMetric, metricFamiliesByName)
		if err != nil {
			log.Print(err.Error())
			continue
		}

		if !emptyMetric {
			logBool, logString := evalThreshold(value, previousValue, previousDiff, scrapeMetric)
			if logBool {
				log.Print(logString)
			}

			sm.previousValues[i] = value
			sm.previousDiffs[i] = value - previousValue
		}
	}
}

// Get value from metric and check if metric is empty.
// Returns: value, emptyMetric, error.
func getValue(metric models.ScrapeEventCondition, metricFamiliesByName map[string]*dto.MetricFamily) (float64, bool, error) {
	if _, exist := metricFamiliesByName[metric.Metric]; !exist {
		return 0, true, nil
	}

	if len(metricFamiliesByName[metric.Metric].Metric) == 0 {
		return 0, true, nil
	}

	if metric.MetricType == "counter" {
		return metricFamiliesByName[metric.Metric].Metric[0].Counter.GetValue(), false, nil
	}
	if metric.MetricType == "gauge" {
		return metricFamiliesByName[metric.Metric].Metric[0].Gauge.GetValue(), false, nil
	}
	return 0, false, fmt.Errorf("metric %q has unrecognized type: %v", metric.Metric, metric.MetricType)
}

func evalThreshold(value float64, previousValue float64, previousDiff float64, scrapeMetric models.ScrapeEventCondition) (bool, string) {
	threshold := scrapeMetric.Threshold
	valueDiff := value - previousValue

	switch scrapeMetric.CheckType {
	case models.ValueThreshold:
		if value > threshold && previousValue <= threshold {
			return true, fmt.Sprintf("Metric %q went above the threshold. Old value: %v New value: %v", scrapeMetric.Metric, previousValue, value)
		}
		if value <= threshold && previousValue > threshold {
			return true, fmt.Sprintf("Metric %q went below the threshold. Old value: %v New value: %v", scrapeMetric.Metric, previousValue, value)
		}
	case models.RateThreshold:
		if valueDiff > threshold && previousDiff <= threshold {
			return true, fmt.Sprintf("Metric rate %q went above the threshold. Old rate: %v New rate: %v", scrapeMetric.Metric, previousDiff, valueDiff)
		}
		if valueDiff <= threshold && previousDiff > threshold {
			return true, fmt.Sprintf("Metric rate %q went below the threshold. Old rate: %v New rate: %v", scrapeMetric.Metric, previousDiff, valueDiff)
		}
	}
	return false, ""
}
