// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/stretchr/testify/require"
)

func TestPipeline_AddCollectors(t *testing.T) {
	pipeline := NewPipeline("foo")
	collector1 := collectors.NewGoCollector()
	collector2 := collectors.NewBuildInfoCollector()

	pipeline.AddCollectors(collector1, collector2)
	require.ElementsMatch(t, pipeline.collectors, []prometheus.Collector{collector1, collector2})
}

func TestPipeline_GetEndpointHandler(t *testing.T) {
	pipeline := NewPipeline("foo")
	handler := pipeline.GetEndpointHandler()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)
}
