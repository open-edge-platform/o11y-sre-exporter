// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

const (
	// config baselines as files.
	pathToTestConfigData   = "../../deployments/sre-exporter/files/configs"
	configFileNameOrch     = "sre-exporter-orch.json"
	configFileNameEdgeNode = "sre-exporter-edge-node.json"

	pathToTestGenCollectorInputData  = "testdata/generic_collector/input"
	pathToTestGenCollectorOutputData = "testdata/generic_collector/output"
	// empty response defined inline.
	metricsEmptyResponse = `
	{
		"status": "success",
		"data": {
		  "resultType": "vector",
		  "result": []
		}
	}
	`
)

type TestPipeline struct {
	registry   *prometheus.Registry
	collectors []prometheus.Collector
}

func NewPipeline() *TestPipeline {
	return &TestPipeline{
		registry: prometheus.NewRegistry(),
	}
}

func (pipeline *TestPipeline) AddCollectors(collectors ...prometheus.Collector) {
	pipeline.registry.MustRegister(collectors...)
	pipeline.collectors = append(pipeline.collectors, collectors...)
}

func (pipeline *TestPipeline) GetEndpointHandler() http.Handler {
	return promhttp.HandlerFor(pipeline.registry, promhttp.HandlerOpts{Registry: pipeline.registry})
}

func readConfigs(t *testing.T, configPath string, configFiles []string) map[string]string {
	configMap := make(map[string]string)
	for _, configFile := range configFiles {
		configMap[configFile] = readFileContents(t, path.Join(configPath, configFile))
	}
	return configMap
}

func readFileContents(t *testing.T, filePath string) string {
	contents, err := os.ReadFile(filePath)
	require.NoError(t, err)
	return string(contents)
}

func createMockHandlerFunc(t *testing.T, queryResponseMap map[string]string, expectedOrgID string,
	returnCode int, latency time.Duration) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(returnCode)
		if returnCode == 200 {
			orgID := req.Header.Get(HeaderXScopeOrgID)
			require.Equal(t, expectedOrgID, orgID)
			require.NoError(t, req.ParseForm())
			if latency != 0 {
				time.Sleep(latency)
			}
			response := metricsEmptyResponse
			queryString := req.Form.Get("query")
			require.NotEmpty(t, queryString)
			// return the response if the query exists as a key, otherwise empty response will be kept
			if value, ok := queryResponseMap[queryString]; ok {
				response = value
			}
			// Send response to be tested
			n, err := rw.Write([]byte(response))
			require.NoError(t, err)
			require.Len(t, response, n)
		}
	}
}

func setupMockServer(t *testing.T, queryResponses map[string]string, org string,
	returnCode int, latency time.Duration) *httptest.Server {
	// load metrics from files
	queryResponseMap := make(map[string]string)
	for query, responseFile := range queryResponses {
		queryResponseMap[query] = readFileContents(t, path.Join(pathToTestGenCollectorInputData, responseFile))
	}
	return httptest.NewServer(createMockHandlerFunc(t, queryResponseMap, org, returnCode, latency))
}

func loadConfig(t *testing.T, configMap map[string]string, key string) models.Configuration {
	var config models.Configuration
	require.NoError(t, json.Unmarshal([]byte(configMap[key]), &config))
	return config
}

func TestCollector(t *testing.T) {
	type sourceMock struct {
		jsonConfig     string
		queryResponses map[string]string
	}
	type args struct {
		sources             []sourceMock
		expectedOutputsFile string
		svrReturnCode       int
		latency             time.Duration
		serverOn            bool
	}
	tests := []struct {
		name      string
		arguments args
	}{
		{
			name: "orch-api-and-edge-node-sunny-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				expectedOutputsFile: "expected_outputs.txt",
				svrReturnCode:       200,
				latency:             0,
				serverOn:            true,
			},
		},
		{
			name: "orch-api-and-edge-node-metrics-error-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle_error.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				expectedOutputsFile: "expected_outputs_error.txt",
				svrReturnCode:       200,
				latency:             0,
				serverOn:            true,
			},
		},
		{
			name: "orch-api-and-edge-node-metrics-incomplete-response-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle_incomplete.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				expectedOutputsFile: "expected_outputs_incomplete.txt",
				svrReturnCode:       200,
				latency:             0,
				serverOn:            true,
			},
		},
		{
			name: "orch-api-and-edge-node-metrics-server-error-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle_incomplete.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				expectedOutputsFile: "expected_outputs_server_error.txt",
				svrReturnCode:       503,
				latency:             0,
				serverOn:            true,
			},
		},
		{
			name: "orch-api-and-edge-node-latency-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				// In expected output we put 100ms latency because of nondeterministic fluctioation of few ms.
				expectedOutputsFile: "expected_outputs_latency.txt",
				svrReturnCode:       200,
				latency:             1 * time.Second,
				serverOn:            true,
			},
		},
		{
			name: "orch-api-and-edge-node-server-not-working-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				expectedOutputsFile: "expected_outputs_server_not_working.txt",
				svrReturnCode:       200,
				latency:             1 * time.Second,
				serverOn:            false,
			},
		},
		{
			name: "orch-api-and-edge-node-metrics-invalid-json-response-scenario",
			arguments: args{
				sources: []sourceMock{
					{
						jsonConfig: configFileNameOrch,
						// query responses baselines as files
						queryResponses: map[string]string{
							"traefik_service_requests_total":                   "orch_traefik_service_requests_total.json",
							"sum by(k8s_node_name) (k8s_node_allocatable_cpu)": "orch_cpu_total_cores.json",
							"sum by(k8s_node_name) (k8s_node_cpu_usage)":       "orch_cpu_used_cores.json",
							"k8s_node_allocatable_memory":                      "orch_memory_total_bytes.json",
							"k8s_node_memory_available":                        "orch_memory_available_bytes.json",
						},
					},
					{
						jsonConfig: configFileNameEdgeNode,
						// query responses baselines as files
						queryResponses: map[string]string{
							"cpu_usage_idle{cpu='cpu-total'}": "edgenode_cpu_usage_idle_invalid.json",
							"disk_used_percent":               "edgenode_disk_used_percent.json",
							"mem_used_percent":                "edgenode_mem_used_percent.json",
							"temp_temp":                       "edgenode_temp_temp.json",
						},
					},
				},
				expectedOutputsFile: "expected_outputs_incomplete.txt",
				svrReturnCode:       200,
				latency:             0,
				serverOn:            true,
			},
		},
	}

	configMap := readConfigs(t, pathToTestConfigData, []string{configFileNameOrch, configFileNameEdgeNode})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ppl := NewPipeline()
			for _, source := range tt.arguments.sources {
				config := loadConfig(t, configMap, source.jsonConfig)
				if tt.arguments.serverOn {
					mockServer := setupMockServer(t, source.queryResponses, config.Source.Org,
						tt.arguments.svrReturnCode, tt.arguments.latency)
					// Close the server when test finishes
					defer mockServer.Close() //nolint:gocritic // we need each server running until we get all metrics at the end of the function
					config.Source.URI = mockServer.URL
				}

				collectors, err := BuildCollectorsFromConfig(&config, "test-customer")
				require.NoError(t, err)
				ppl.AddCollectors(collectors...)
			}
			handler := ppl.GetEndpointHandler()
			response := assert.HTTPBody(handler.ServeHTTP, http.MethodGet, "/metrics", nil)

			expectedOutputs := strings.Split(readFileContents(t, path.Join(pathToTestGenCollectorOutputData, tt.arguments.expectedOutputsFile)), "\n")
			for _, substr := range expectedOutputs {
				require.Containsf(t, response, substr,
					"The server response does not contain the expected substring: \n%s\n\n==== Response received below ====\n%s\n",
					substr, response)
			}
		})
	}
}
