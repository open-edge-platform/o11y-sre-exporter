// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import "github.com/open-edge-platform/o11y-sre-exporter/internal/models"

// hardcoded metrics for now

var metric1 = models.ScrapeEventCondition{
	Metric:     "otelcol_exporter_send_failed_metric_points",
	CheckType:  models.RateThreshold,
	Threshold:  0,
	MetricType: "counter",
}

var metric2 = models.ScrapeEventCondition{
	Metric:     "otelcol_exporter_sent_metric_points",
	CheckType:  models.RateThreshold,
	Threshold:  0,
	MetricType: "counter",
}

var metric3 = models.ScrapeEventCondition{
	Metric:     "otelcol_receiver_accepted_metric_points",
	CheckType:  models.RateThreshold,
	Threshold:  0,
	MetricType: "counter",
}
var metric4 = models.ScrapeEventCondition{
	Metric:     "otelcol_receiver_refused_metric_points",
	CheckType:  models.RateThreshold,
	Threshold:  0,
	MetricType: "counter",
}
var metric5 = models.ScrapeEventCondition{
	Metric:     "otelcol_process_memory_rss",
	CheckType:  models.ValueThreshold,
	Threshold:  2147483648, // 2147483648 bytes = 2GiB
	MetricType: "gauge",
}

var scrapedOtelMetricsList = []models.ScrapeEventCondition{metric1, metric2, metric3, metric4, metric5}
