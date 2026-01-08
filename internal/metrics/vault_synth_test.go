// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	pathToTestVaultOutputData = "testdata/vault_collector/output"
	pathToTestVaultInputData  = "testdata/vault_collector/input"
	delta                     = float64(0.0001)
	vaultPodsNamespace        = "orch-platform"
)

func TestGetVaultStatus(t *testing.T) {
	t.Run("bad url", func(t *testing.T) {
		gaugeMetric, err := getVaultPodStatus("4$@3us", "test")
		require.ErrorContains(t, err, "error parsing URL")
		require.Zero(t, gaugeMetric)
	})
	t.Run("error getting url", func(t *testing.T) {
		gaugeMetric, err := getVaultPodStatus("localhost", "90000")
		require.ErrorContains(t, err, "error getting")
		require.Zero(t, gaugeMetric)
	})

	t.Run("bad json", func(t *testing.T) {
		testURL := newHTTPTestServer(t, http.StatusOK, "test")

		gaugeMetric, err := getVaultPodStatus(testURL.Hostname(), testURL.Port())
		require.ErrorContains(t, err, "error during unmarshal")
		require.Zero(t, gaugeMetric)
	})

	t.Run("valid json - status unknown", func(t *testing.T) {
		json, err := os.ReadFile(path.Join(pathToTestVaultInputData, "unknown.json"))
		require.NoError(t, err)
		testURL := newHTTPTestServer(t, http.StatusOK, string(json))

		gaugeMetric, err := getVaultPodStatus(testURL.Hostname(), testURL.Port())
		require.NoError(t, err)
		require.InDelta(t, float64(unknown), float64(gaugeMetric), delta)
	})

	t.Run("valid json - status Sealed", func(t *testing.T) {
		json, err := os.ReadFile(path.Join(pathToTestVaultInputData, "sealed.json"))
		require.NoError(t, err)
		testURL := newHTTPTestServer(t, http.StatusOK, string(json))

		gaugeMetric, err := getVaultPodStatus(testURL.Hostname(), testURL.Port())
		require.NoError(t, err)
		require.InDelta(t, float64(sealed), float64(gaugeMetric), delta)
	})

	t.Run("valid json - status Standby", func(t *testing.T) {
		json, err := os.ReadFile(path.Join(pathToTestVaultInputData, "standby.json"))
		require.NoError(t, err)
		testURL := newHTTPTestServer(t, http.StatusOK, string(json))

		gaugeMetric, err := getVaultPodStatus(testURL.Hostname(), testURL.Port())
		require.NoError(t, err)
		require.InDelta(t, float64(standby), float64(gaugeMetric), delta)
	})

	t.Run("valid json - status Ready", func(t *testing.T) {
		json, err := os.ReadFile(path.Join(pathToTestVaultInputData, "ready.json"))
		require.NoError(t, err)
		testURL := newHTTPTestServer(t, http.StatusOK, string(json))

		gaugeMetric, err := getVaultPodStatus(testURL.Hostname(), testURL.Port())
		require.NoError(t, err)
		require.InDelta(t, float64(ready), float64(gaugeMetric), delta)
	})
}

func TestCollect(t *testing.T) {
	namespace := VaultMetricNamespace
	customer := "cs"
	t.Run("empty vault URI", func(t *testing.T) {
		clientSet := fake.NewClientset()
		collector := NewVaultSynthCollector(clientSet, nil, vaultPodsNamespace, DefaultPodPort, customer)
		expected, err := os.ReadFile(path.Join(pathToTestVaultOutputData, "empty"))
		require.NoError(t, err)

		err = testutil.CollectAndCompare(collector, bytes.NewReader(expected))
		require.NoError(t, err)
	})
	t.Run("no pods in k8s", func(t *testing.T) {
		clientSet := fake.NewClientset()
		vaultPodsName := []string{"vault-1"}
		collector := NewVaultSynthCollector(clientSet, vaultPodsName, vaultPodsNamespace, DefaultPodPort, customer)
		expected, err := os.ReadFile(path.Join(pathToTestVaultOutputData, "warnings"))
		require.NoError(t, err)

		err = testutil.CollectAndCompare(collector, bytes.NewReader(expected))
		require.NoError(t, err)
	})

	t.Run("1 pod in k8s, but can't get vault IP", func(t *testing.T) {
		testURL := newHTTPTestServer(t, http.StatusInternalServerError, "")
		vaultPodsName := []string{"vault-1"}
		p := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vaultPodsName[0],
				Namespace: vaultPodsNamespace,
			},
			Status: corev1.PodStatus{
				PodIP: testURL.Hostname(),
			},
		}
		clientSet := fake.NewClientset(p)
		collector := NewVaultSynthCollector(clientSet, vaultPodsName, vaultPodsNamespace, testURL.Port(), customer)
		expected, err := os.ReadFile(path.Join(pathToTestVaultOutputData, "warnings"))
		require.NoError(t, err)

		checkMetrics(t, 4, collector, namespace, expected)
	})

	t.Run("1 pod in k8s", func(t *testing.T) {
		json, err := os.ReadFile(path.Join(pathToTestVaultInputData, "sealed.json"))
		require.NoError(t, err)
		testURL := newHTTPTestServer(t, http.StatusOK, string(json))
		vaultPodsName := []string{"vault-1"}
		p := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vaultPodsName[0],
				Namespace: vaultPodsNamespace,
			},
			Status: corev1.PodStatus{
				PodIP: testURL.Hostname(),
			},
		}
		clientSet := fake.NewClientset(p)
		collector := NewVaultSynthCollector(clientSet, vaultPodsName, vaultPodsNamespace, testURL.Port(), customer)
		expected, err := os.ReadFile(path.Join(pathToTestVaultOutputData, "ok_1_pod"))
		require.NoError(t, err)

		checkMetrics(t, 5, collector, namespace, expected)
	})

	t.Run("2 pods in k8s", func(t *testing.T) {
		json, err := os.ReadFile(path.Join(pathToTestVaultInputData, "sealed.json"))
		require.NoError(t, err)
		testURL := newHTTPTestServer(t, http.StatusOK, string(json))
		vaultPodsName := []string{"vault-1", "vault-2"}

		pods := []runtime.Object{
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vaultPodsName[0],
					Namespace: vaultPodsNamespace,
				},
				Status: corev1.PodStatus{
					PodIP: testURL.Hostname(),
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vaultPodsName[1],
					Namespace: vaultPodsNamespace,
				},
				Status: corev1.PodStatus{
					PodIP: testURL.Hostname(),
				},
			},
		}
		clientSet := fake.NewClientset(pods...)

		collector := NewVaultSynthCollector(clientSet, vaultPodsName, vaultPodsNamespace, testURL.Port(), customer)
		expected, err := os.ReadFile(path.Join(pathToTestVaultOutputData, "ok_2_pods"))
		require.NoError(t, err)

		checkMetrics(t, 6, collector, namespace, expected)
	})
}

func newHTTPTestServer(t *testing.T, statusCode int, response string) *url.URL {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, basePath, r.URL.Path[1:])
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(response))
		require.NoError(t, err)
	}))

	testURL, err := url.ParseRequestURI(server.URL)
	require.NoError(t, err)

	t.Cleanup(func() {
		server.Close()
	})

	return testURL
}

func checkMetrics(t *testing.T, metricsNum int, collector prometheus.Collector, namespace string, expected []byte) {
	// checking overall metrics num
	require.Equal(t, metricsNum, testutil.CollectAndCount(collector, nil...))

	// Check vault_status_query_latency_milliseconds metric
	require.Equal(t, 1, testutil.CollectAndCount(collector,
		[]string{prometheus.BuildFQName(namespace, VaultStatusSubSystemName, vaultStatusQueryLatencyName)}...))

	// Ignoring vault_status_query_latency_milliseconds metric (can't compare)
	expectedMetrics := []string{
		prometheus.BuildFQName(namespace, VaultMonitorSubSystemName, VaultStatusSubSystemName),
		prometheus.BuildFQName(namespace, VaultStatusSubSystemName, vaultStatusUpName),
		prometheus.BuildFQName(namespace, VaultStatusSubSystemName, vaultStatusWarningsName),
		prometheus.BuildFQName(namespace, VaultStatusSubSystemName, vaultStatusQuerySamplesName),
	}

	require.NoError(t, testutil.CollectAndCompare(collector, bytes.NewReader(expected), expectedMetrics...))
}
