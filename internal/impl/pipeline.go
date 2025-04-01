// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Pipeline struct {
	registry   *prometheus.Registry
	collectors []prometheus.Collector
	namespace  string
}

// Namespace must be unique for every pipeline.
func NewPipeline(namespace string) *Pipeline {
	return &Pipeline{
		registry:  prometheus.NewRegistry(),
		namespace: namespace,
	}
}

func (pipeline *Pipeline) AddCollectors(collectors ...prometheus.Collector) {
	pipeline.registry.MustRegister(collectors...)
	pipeline.collectors = append(pipeline.collectors, collectors...)
}

func (pipeline *Pipeline) UnregisterCollectors() error {
	for _, collector := range pipeline.collectors {
		if ok := pipeline.registry.Unregister(collector); !ok {
			return fmt.Errorf("could not unregister collector: '%v'", collector)
		}
	}
	pipeline.collectors = nil
	return nil
}

func (pipeline *Pipeline) GetEndpointHandler() http.Handler {
	return promhttp.HandlerFor(pipeline.registry, promhttp.HandlerOpts{Registry: pipeline.registry})
}

func (pipeline *Pipeline) GetNamespace() string {
	return pipeline.namespace
}
