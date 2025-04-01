// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

const jsonPatternEmptyLabels = `
{
  "namespace": "orch",
  "collectors": [
    {
      "name": "NodeCollector",
      "enabled": true,
      "metrics": [
        {
          "name": "nodeCpuTotalQuery",
          "query": "sum by (instance,mode) (node_cpu_seconds_total)",
          "id": "cpu_total",
          "help": "CPU seconds per mode by mode (idle,system,steal etc)",
          "labels": [
            "instance",
            "mode"
          ]
        },
        {
          "name":"nodeMemTotalQuery",
          "query":"node_memory_MemTotal_bytes",
          "id":"memory_MemTotal_bytes",
          "help":"Total memory on node",
          "labels": [
            "instance"
          ]
        },
        {
          "name": "nodeMemAvailQuery",
          "query": "node_memory_MemAvailable_bytes",
          "id": "memory_MemAvail_byes",
          "help": "Current available memory on node",
          "labels": [
            "instance"
          ]
        }
      ]
    },
    {
      "name": "api",
      "enabled": true,
      "metrics": [
        {
          "name": "traefikRequestsTotalQuery",
          "query": "traefik_service_requests_total",
          "id": "requests_all",
          "help": "The total count of HTTP request processed",
          "labels": [
            "code",
            "exported_service",
            "method",
            "protocol",
            "instance",
            "namespace",
            "pod"
          ],
          "destLabels": [
            "status",
            "target_service",
            "method",
            "protocol",
            "gw_instance",
            "gw_namespace",
            "gw_pod"
          ]
        },
        {
          "name": "traefikDurationQuery",
          "query": "traefik_service_request_duration_seconds_bucket",
          "id": "request_latency_seconds_all",
          "help": "Histogram of HTTP request latencies",
          "labels": [
            "code",
            "exported_service",
            "method",
            "protocol",
            "instance",
            "namespace",
            "pod",
            "le"
          ],
          "destLabels": [
            "status",
            "target_service",
            "method",
            "protocol",
            "gw_instance",
            "gw_namespace",
            "gw_pod",
            "le"
          ]
        }
      ]
    }
  ]
}`

const jsonPatternEmptyLabelsExpected = `
{
  "namespace": "orch",
  "collectors": [
    {
      "name": "NodeCollector",
      "enabled": true,
      "metrics": [
        {
          "name": "nodeCpuTotalQuery",
          "query": "sum by (instance,mode) (node_cpu_seconds_total)",
          "id": "cpu_total",
          "help": "CPU seconds per mode by mode (idle,system,steal etc)",
          "labels": [
            "instance",
            "mode"
          ],
		  "destLabels": [
            "instance",
            "mode"
          ]
        },
        {
          "name":"nodeMemTotalQuery",
          "query":"node_memory_MemTotal_bytes",
          "id":"memory_MemTotal_bytes",
          "help":"Total memory on node",
          "labels": [
            "instance"
          ],
		  "destLabels": [
            "instance"
          ]
        },
        {
          "name": "nodeMemAvailQuery",
          "query": "node_memory_MemAvailable_bytes",
          "id": "memory_MemAvail_byes",
          "help": "Current available memory on node",
          "labels": [
            "instance"
          ],
		  "destLabels": [
            "instance"
          ]
        }
      ]
    },
    {
      "name": "api",
      "enabled": true,
      "metrics": [
        {
          "name": "traefikRequestsTotalQuery",
          "query": "traefik_service_requests_total",
          "id": "requests_all",
          "help": "The total count of HTTP request processed",
          "labels": [
            "code",
            "exported_service",
            "method",
            "protocol",
            "instance",
            "namespace",
            "pod"
          ],
          "destLabels": [
            "status",
            "target_service",
            "method",
            "protocol",
            "gw_instance",
            "gw_namespace",
            "gw_pod"
          ]
        },
        {
          "name": "traefikDurationQuery",
          "query": "traefik_service_request_duration_seconds_bucket",
          "id": "request_latency_seconds_all",
          "help": "Histogram of HTTP request latencies",
          "labels": [
            "code",
            "exported_service",
            "method",
            "protocol",
            "instance",
            "namespace",
            "pod",
            "le"
          ],
          "destLabels": [
            "status",
            "target_service",
            "method",
            "protocol",
            "gw_instance",
            "gw_namespace",
            "gw_pod",
            "le"
          ]
        }
      ]
    }
  ]
}`

func Test_readConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Run("errNotExist", func(t *testing.T) {
		var errExpected *fs.PathError
		configFilePath := filepath.Join(tmpDir, "nonexistsconfig")

		out, err := readConfig(&configFilePath)
		require.ErrorAs(t, err, &errExpected)
		require.Empty(t, out)
	})
	t.Run("expectedBehavior", func(t *testing.T) {
		value := []byte("test")
		configFilePath := filepath.Join(tmpDir, "exist")
		err := os.WriteFile(configFilePath, value, 0640)
		require.NoError(t, err)

		out, err := readConfig(&configFilePath)
		require.NoError(t, err)
		require.Equal(t, out, value)
	})
}

func Test_InitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Run("errNotExist", func(t *testing.T) {
		var errExpected *fs.PathError
		configFilePath := filepath.Join(tmpDir, "nonexistsconfig")

		out, hash, err := InitConfig(&configFilePath)
		require.ErrorAs(t, err, &errExpected)
		require.Empty(t, out)
		require.Empty(t, hash)
	})

	t.Run("unknown format", func(t *testing.T) {
		var errExpected *json.SyntaxError
		configFilePath := filepath.Join(tmpDir, "config_unknown_format")
		err := os.WriteFile(configFilePath, []byte("brokenJson"), 0640)
		require.NoError(t, err)

		out, hash, err := InitConfig(&configFilePath)
		require.ErrorAs(t, err, &errExpected)
		require.Empty(t, out)
		require.Empty(t, hash)
	})

	t.Run("empty json", func(t *testing.T) {
		configFilePath := filepath.Join(tmpDir, "config_empty")
		err := os.WriteFile(configFilePath, []byte("{}"), 0640)
		require.NoError(t, err)

		out, hash, err := InitConfig(&configFilePath)
		require.NoError(t, err)
		require.Empty(t, out)
		require.Equal(t, "0fbec9a36052522bf90039729ed1b362eca392d5065a1b7251e67421323957dd", hash)
	})

	t.Run("valid json", func(t *testing.T) {
		configFilePath := filepath.Join(tmpDir, "config_valid")
		err := os.WriteFile(configFilePath, []byte(jsonPatternEmptyLabels), 0640)
		require.NoError(t, err)

		var configurationExpected models.Configuration
		err = json.Unmarshal([]byte(jsonPatternEmptyLabelsExpected), &configurationExpected)
		require.NoError(t, err)

		out, hash, err := InitConfig(&configFilePath)
		require.NoError(t, err)
		require.Equal(t, &configurationExpected, out)
		require.Equal(t, "372a1dd4556468c7237d55cac8e46bbb2a1f0c0485db158e0730bad9774fd39b", hash)
	})
}

const sampleConfig = `
{
  "namespace": "orch_edgenode",
  "source": {
    "queryURI": "http://localhost:8181/prometheus",
    "mimirOrg": "eae6c20c-4fb9-449f-82d0-b290e10595d4"
  },
  "collectors": [
    {
      "name": "env",
      "enabled": true,
      "metrics": [
        {
          "name": "Temperature Celsius",
          "enabled": false,
          "query": "temp_temp",
          "id": "temp",
          "help": "Temperature of a sensor on Edge Node host [Celsius]",
          "description": null,
          "labels": [
            "host",
            "hostGuid",
            "projectId",
            "sensor"
          ],
          "destLabels": null,
          "Type": "Gauge"
        }
      ]
    },
    {
      "name": "mem",
      "enabled": true,
      "metrics": [
        {
          "name": "Memory Used Percent",
          "enabled": false,
          "query": "mem_used_percent",
          "id": "used_percent",
          "help": "Percentage of used memory vs total memory on Edge Node host",
          "description": null,
          "labels": [
            "host",
            "hostGuid",
            "projectId"
          ],
          "destLabels": null,
          "Type": "Gauge"
        }
      ]
    },
    {
      "name": "disk",
      "enabled": true,
      "metrics": [
        {
          "name": "Disk Used Percent",
          "enabled": false,
          "query": "disk_used_percent",
          "id": "used_percent",
          "help": "Percentage of used vs total available space on a disk of Edge Node host",
          "description": null,
          "labels": [
            "host",
            "hostGuid",
            "projectId",
            "device",
            "path",
            "mode"
          ],
          "destLabels": null,
          "Type": "Gauge"
        }
      ]
    },
    {
      "name": "cpu",
      "enabled": true,
      "metrics": [
        {
          "name": "Total CPU Idle Percent",
          "enabled": false,
          "query": "cpu_usage_idle{cpu='cpu-total'}",
          "id": "idle_percent",
          "help": "Percentage of idle vs total CPU cycles on Edge Node host",
          "description": null,
          "labels": [
            "host",
            "hostGuid",
            "projectId"
          ],
          "destLabels": null,
          "Type": "Gauge"
        }
      ]
    }
  ]
}
`

func Test_GetConfigHash(t *testing.T) {
	var config1 models.Configuration
	var err error
	require.NoError(t, json.Unmarshal([]byte(sampleConfig), &config1))

	t.Run("same content", func(t *testing.T) {
		var config2 models.Configuration
		var compacted bytes.Buffer
		require.NoError(t, json.Compact(&compacted, []byte(sampleConfig)))
		require.NoError(t, json.Unmarshal(compacted.Bytes(), &config2))

		var hash1, hash2 string
		hash1, err = GetConfigHash(&config1)
		require.NoError(t, err)
		hash2, err = GetConfigHash(&config2)
		require.NoError(t, err)

		require.Equal(t, hash1, hash2)
	})
	t.Run("different content", func(t *testing.T) {
		var config2 = config1
		config1.Source.Org = "eae6c20c-4fb9-449f-82d0-b290e10595d4|b5a69fa4-57c4-4bee-b738-4935b861f1c4"

		var hash1, hash2 string
		hash1, err = GetConfigHash(&config1)
		require.NoError(t, err)
		hash2, err = GetConfigHash(&config2)
		require.NoError(t, err)

		require.NotEqual(t, hash1, hash2)
	})
}
