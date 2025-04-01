// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRegisterPipeline(t *testing.T) {
	pipeline := NewPipeline("foo")
	listenAddress := ":45566"
	manager := NewPipelineManager(&listenAddress)
	manager.RegisterPipeline("/metrics", pipeline)
	require.ElementsMatch(t, manager.pipelines, []*Pipeline{pipeline})
}

func TestStart(t *testing.T) {
	listenAddress := ":45566"
	manager := NewPipelineManager(&listenAddress)
	manager.RegisterHealthCheck()
	go func() {
		if err := manager.Start(); !errors.Is(err, http.ErrServerClosed) {
			t.Logf("Server error: %v", err)
			t.Fail()
		}
	}()
	time.Sleep(1 * time.Second)

	// Test health check
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	manager.routerSwapper.router.ServeHTTP(w, req)
	require.Equal(t, "OK", w.Body.String())
}
