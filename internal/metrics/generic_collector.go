// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"log"

	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

// we sum by instance because we don't care about individual core/thread just
// the total for each mode and instance combo

type GenericCollector struct {
	v1api                    promv1.API
	namespace                string
	collector                *models.Collector
	constLabels              prometheus.Labels
	up                       *prometheus.Desc
	warnings                 *prometheus.Desc
	querySamples             *prometheus.Desc
	queryLatencyMilliseconds *prometheus.Desc
}

const (
	constLabelService  = "service"
	constLabelCustomer = "customer"
)

var ConstLabels = [...]string{constLabelService, constLabelCustomer}

func BuildCollectorsFromConfig(config *models.Configuration, customer string) ([]prometheus.Collector, error) {
	client, err := api.NewClient(api.Config{
		Address:      config.Source.URI,
		RoundTripper: newMimirRoundTripper(&config.Source.Org),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	v1api := promv1.NewAPI(client)

	constLabels := prometheus.Labels{
		constLabelService:  config.Namespace,
		constLabelCustomer: customer,
	}

	collectors := config.Collectors
	var parsedCollectors []prometheus.Collector
	for i := range collectors {
		if collectors[i].Enabled {
			collector := NewGenericCollector(v1api, config.Namespace, constLabels, &collectors[i])
			parsedCollectors = append(parsedCollectors, prometheus.Collector(collector))
		}
	}
	return parsedCollectors, nil
}

func NewGenericCollector(v1api promv1.API, namespace string,
	constLabels prometheus.Labels, collector *models.Collector) *GenericCollector {
	log.Printf("NewGenericCollector(%v, %s, %v, %v)", v1api, namespace, constLabels, collector)
	for i := 0; i < len(collector.Metrics); i++ {
		thisMetric := &collector.Metrics[i]
		if len(thisMetric.DestLabels) == 0 {
			thisMetric.Description = prometheus.NewDesc(prometheus.BuildFQName(namespace, collector.Name, thisMetric.ID),
				thisMetric.Help, thisMetric.Labels, constLabels)
		} else {
			thisMetric.Description = prometheus.NewDesc(prometheus.BuildFQName(namespace, collector.Name, thisMetric.ID),
				thisMetric.Help, thisMetric.DestLabels, constLabels)
		}
	}

	genColl := &GenericCollector{
		v1api:       v1api,
		namespace:   namespace,
		collector:   collector,
		constLabels: constLabels,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, collector.Name, "up"),
			"Were all the last backend queries successful",
			nil, constLabels),
		warnings: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, collector.Name, "warnings"),
			"How many warnings did the last queries generate",
			nil, constLabels),
		querySamples: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, collector.Name, "query_samples"),
			"How many samples did the last queries generate",
			nil, constLabels),
		queryLatencyMilliseconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, collector.Name, "query_latency_milliseconds"),
			"How long did it take to perform the slowest query",
			nil, constLabels),
	}
	return genColl
}

func (genColl *GenericCollector) Describe(descs chan<- *prometheus.Desc) {
	for i := 0; i < len(genColl.collector.Metrics); i++ {
		thisMetric := &genColl.collector.Metrics[i]
		descs <- thisMetric.Description
	}
}

func (genColl *GenericCollector) Collect(metrics chan<- prometheus.Metric) {
	stats := CollectStats{
		Up:            true,
		LatencyMillis: 0,
		Samples:       0,
		Warnings:      0,
	}
	for i := 0; i < len(genColl.collector.Metrics); i++ {
		metric := &genColl.collector.Metrics[i]
		singleStat := processCounterQuery(&metric.Query, &metric.Labels,
			metrics, genColl.v1api, genColl.collector.Metrics[i].Description,
			genColl.collector.Metrics[i].Type)
		reconcileStats(&stats, &singleStat)
	}
	var upf float64
	if stats.Up {
		upf = 1
	}
	metrics <- prometheus.MustNewConstMetric(
		genColl.up, prometheus.GaugeValue, upf,
	)
	metrics <- prometheus.MustNewConstMetric(
		genColl.queryLatencyMilliseconds, prometheus.GaugeValue, float64(stats.LatencyMillis),
	)
	metrics <- prometheus.MustNewConstMetric(
		genColl.warnings, prometheus.GaugeValue, float64(stats.Warnings),
	)
	metrics <- prometheus.MustNewConstMetric(
		genColl.querySamples, prometheus.GaugeValue, float64(stats.Samples),
	)
}

func reconcileStats(mainStats *CollectStats, singleStat *CollectStats) {
	mainStats.Up = mainStats.Up && singleStat.Up
	if singleStat.LatencyMillis > mainStats.LatencyMillis {
		mainStats.LatencyMillis = singleStat.LatencyMillis
	}
	mainStats.Samples += singleStat.Samples
	mainStats.Warnings += singleStat.Warnings
}
