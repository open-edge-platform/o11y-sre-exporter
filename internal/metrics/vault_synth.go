// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

type vaultStatus float64

const (
	// vault status const.
	unknown vaultStatus = -1.0
	ready   vaultStatus = 0.0
	sealed  vaultStatus = 1.0
	standby vaultStatus = 2.0

	// vault const.
	basePath = "v1/sys/health"

	// Kubernetes const.
	DefaultPodPort = "8200"

	// Metrics const.
	VaultMetricNamespace        = "orch"
	VaultMonitorSubSystemName   = "vault_monitor"
	VaultStatusSubSystemName    = "vault_status"
	vaultStatusUpName           = "up"
	vaultStatusWarningsName     = "warnings"
	vaultStatusQuerySamplesName = "query_samples"
	vaultStatusQueryLatencyName = "query_latency_milliseconds"
	VaultStatusDescription      = "The current status of vault instance ready:0 , sealed:1 , standby:2"
	// Labels const.
	VaultInstanceLabelName = "k8s_pod_name"
)

type vaultSynthCollector struct {
	k8sCli             k8s.Interface
	namespace          string
	vaultPodsNamespace string
	vaultPodsName      []string
	podPort            string

	vaultInstanceStatus            *prometheus.Desc
	upMetric                       *prometheus.Desc
	warningsMetric                 *prometheus.Desc
	querySamplesMetric             *prometheus.Desc
	queryLatencyMillisecondsMetric *prometheus.Desc
}

type vaultStatusResponse struct {
	Initialized                bool   `json:"initialized"`
	Sealed                     bool   `json:"sealed"`
	Standby                    bool   `json:"standby"`
	PerformanceStandby         bool   `json:"performance_standby"`
	ReplicationPerformanceMode string `json:"replication_performance_mode"`
	ReplicationDrMode          string `json:"replication_dr_mode"`
	ServerTimeUtc              uint   `json:"server_time_utc"`
	Version                    string `json:"version"`
	ClusterName                string `json:"cluster_name"`
	ClusterID                  string `json:"cluster_id"`
}

func NewVaultSynthCollector(k8sCli k8s.Interface, vaultPodsName []string, vaultPodsNamespace, podPort, customer string) prometheus.Collector {
	constLabels := prometheus.Labels{
		constLabelService:  VaultMetricNamespace,
		constLabelCustomer: customer,
	}

	collector := &vaultSynthCollector{
		podPort:            podPort,
		k8sCli:             k8sCli,
		vaultPodsNamespace: vaultPodsNamespace,
		vaultPodsName:      vaultPodsName,
		namespace:          VaultMetricNamespace,
		vaultInstanceStatus: prometheus.NewDesc(
			prometheus.BuildFQName(VaultMetricNamespace, VaultMonitorSubSystemName, VaultStatusSubSystemName),
			VaultStatusDescription, []string{VaultInstanceLabelName},
			constLabels),
		upMetric: prometheus.NewDesc(
			prometheus.BuildFQName(VaultMetricNamespace, VaultStatusSubSystemName, vaultStatusUpName),
			"Were all the last backend queries successful",
			nil, constLabels),
		warningsMetric: prometheus.NewDesc(
			prometheus.BuildFQName(VaultMetricNamespace, VaultStatusSubSystemName, vaultStatusWarningsName),
			"How many warnings did the last queries generate",
			nil, constLabels),
		querySamplesMetric: prometheus.NewDesc(
			prometheus.BuildFQName(VaultMetricNamespace, VaultStatusSubSystemName, vaultStatusQuerySamplesName),
			"How many samples did the last queries generate",
			nil, constLabels),
		queryLatencyMillisecondsMetric: prometheus.NewDesc(
			prometheus.BuildFQName(VaultMetricNamespace, VaultStatusSubSystemName, vaultStatusQueryLatencyName),
			"How long did it take to perform the slowest query",
			nil, constLabels),
	}

	return prometheus.Collector(collector)
}

func (c *vaultSynthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vaultInstanceStatus
	ch <- c.upMetric
	ch <- c.warningsMetric
	ch <- c.querySamplesMetric
	ch <- c.queryLatencyMillisecondsMetric
}

func (c *vaultSynthCollector) Collect(metrics chan<- prometheus.Metric) {
	ctx := context.Background()
	stats := CollectStats{Up: true}
	if len(c.vaultPodsName) == 0 {
		stats.Up = false
	}

	for _, vaultPodName := range c.vaultPodsName {
		start := time.Now()
		pod, err := c.k8sCli.CoreV1().Pods(c.vaultPodsNamespace).Get(ctx, vaultPodName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Can't get pod ip for %q: %v", vaultPodName, err)
			stats.Warnings++
			stats.Up = false
			continue
		}

		vaultInstStatus, err := getVaultPodStatus(pod.Status.PodIP, c.podPort)
		if err != nil {
			log.Printf("Can't get vault instance for %q: %v", vaultPodName, err)
			stats.Warnings++
			stats.Up = false
			continue
		}
		elapsed := time.Since(start)

		// Update stats and vault instance metric
		stats.Samples = stats.Samples + 1
		stats.LatencyMillis = max(stats.LatencyMillis, elapsed.Milliseconds())
		metrics <- prometheus.MustNewConstMetric(c.vaultInstanceStatus, prometheus.GaugeValue, float64(vaultInstStatus), vaultPodName)
	}

	var upf float64
	if stats.Up {
		upf = 1
	}

	metrics <- prometheus.MustNewConstMetric(c.upMetric, prometheus.GaugeValue, upf)
	metrics <- prometheus.MustNewConstMetric(c.queryLatencyMillisecondsMetric, prometheus.GaugeValue, float64(stats.LatencyMillis))
	metrics <- prometheus.MustNewConstMetric(c.warningsMetric, prometheus.GaugeValue, float64(stats.Warnings))
	metrics <- prometheus.MustNewConstMetric(c.querySamplesMetric, prometheus.GaugeValue, float64(stats.Samples))
}

// Func will return the status of provided vault pod instance.
func getVaultPodStatus(podIP, podPort string) (vaultStatus, error) {
	addr := net.JoinHostPort(podIP, podPort)
	urlRaw := fmt.Sprintf("http://%s/%s", addr, basePath)
	urlPod, err := url.Parse(urlRaw)
	if err != nil {
		return 0, fmt.Errorf("error parsing URL %q: %w", urlRaw, err)
	}

	resp, err := http.Get(urlPod.String())
	if err != nil {
		return 0, fmt.Errorf("error getting %q: %w", urlPod.String(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var healthResponse vaultStatusResponse
	err = json.Unmarshal(body, &healthResponse)
	if err != nil {
		return 0, fmt.Errorf("error during unmarshal: HTTP code: %q: error: %w", resp.Status, err)
	}

	// Update status
	status := unknown
	if healthResponse.Standby {
		status = standby
	}
	if healthResponse.Initialized {
		status = ready
	}
	if healthResponse.Sealed {
		status = sealed
	}

	return status, nil
}
