// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"log"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

type CollectStats struct {
	Up            bool
	LatencyMillis int64
	Samples       int
	Warnings      int
}

func processCounterQuery(query *string, sourceLabels *[]string, metrics chan<- prometheus.Metric,
	v1api promv1.API, desc *prometheus.Desc, metricType string) CollectStats {
	stats := CollectStats{}
	start := time.Now()
	timeout := 5 * time.Second
	ctx := context.Background()
	result, warns, err := v1api.Query(ctx, *query, time.Now(), promv1.WithTimeout(timeout))
	stats.LatencyMillis = time.Since(start).Milliseconds()

	if err != nil {
		// TODO: use official log library
		log.Printf("Error querying Prometheus: %v\n", err)
		stats.Up = false
		return stats
	}

	stats.Up = true
	stats.Warnings = len(warns)

	if len(warns) > 0 {
		// TODO: use official log library
		log.Printf("Warnings: %v\n", warns)
	}

	if result == nil {
		stats.Samples = 0
		return stats
	}

	stats.Samples = len(result.(model.Vector))

	// Return samples as-is, but rename labels.
	for _, sample := range result.(model.Vector) {
		destLabelValues := make([]string, len(*sourceLabels))
		for i, sourceLabel := range *sourceLabels {
			destLabelValues[i] = string(sample.Metric[model.LabelName(sourceLabel)])
		}
		// timestamp == 0 is probably not valid timestamp eg. response without value field
		if sample.Timestamp == 0 {
			stats.Up = false
			continue
		}
		// Histogram, add le label as the last one.
		var metric prometheus.Metric
		switch metricType {
		case "Counter":
			metric = prometheus.MustNewConstMetric(
				desc, prometheus.CounterValue,
				float64(sample.Value), destLabelValues...)
		case "Gauge":
			metric = prometheus.MustNewConstMetric(
				desc, prometheus.GaugeValue,
				float64(sample.Value), destLabelValues...)
		default:
			log.Printf("Warning: skipping metric of unknown type: %q", metricType)
			continue
		}
		metrics <- metric
	}
	return stats
}
