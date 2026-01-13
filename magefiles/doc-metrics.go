// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//go:build mage

package main

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/metrics"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

type metricSpecConfig struct {
	metricConfigPath     string
	metricBaseline       string
	doNotValidateMetrics []string
}

type metricDescriptor struct {
	name           string
	metricType     string
	description    string
	constantLabels []string
	variableLabels []string
	query          string
}

const (
	docPath              = "docs/exported-metrics-spec.md"
	versionFilePath      = "VERSION"
	columnCount          = 6
	orchMetricsTitle     = "1. Exported Orchestrator Metrics"
	edgeNodeMetricsTitle = "2. Exported Edge Node Metrics"
	vaultMetricsTitle    = "3. Exported Vault Metrics"
	vaultMetricType      = "Gauge"
)

var (
	rootPath      = "."
	defaultHeader = [columnCount]string{"Name", "Type", "Description", "Constant labels", "Variable labels", "Query"}
	// TODO: make it better (no all characters are allowed in metric / label names).
	re = regexp.MustCompile(`^(.+)\{(.+)}`)
)

var metricSpecs = map[string]metricSpecConfig{
	orchMetricsTitle: {
		metricConfigPath:     "deployments/sre-exporter/files/configs/sre-exporter-orch.json",
		metricBaseline:       "internal/metrics/testdata/generic_collector/output/expected_outputs.txt",
		doNotValidateMetrics: []string{"orch_IstioCollector_istio_requests", "orch_api_request_latency_seconds_all"},
	},
	edgeNodeMetricsTitle: {
		metricConfigPath:     "deployments/sre-exporter/files/configs/sre-exporter-edge-node.json",
		metricBaseline:       "internal/metrics/testdata/generic_collector/output/expected_outputs.txt",
		doNotValidateMetrics: []string{},
	},
	vaultMetricsTitle: {
		metricConfigPath:     "",
		metricBaseline:       "internal/metrics/testdata/vault_collector/output/ok_2_pods",
		doNotValidateMetrics: []string{},
	},
}

func makeRow(desc metricDescriptor) tableRow {
	constStr := strings.Join(desc.constantLabels, ", ")
	varStr := strings.Join(desc.variableLabels, ", ")
	row := tableRow{desc.name, desc.metricType, desc.description, constStr, varStr, desc.query}
	return row
}

func buildTableFromDescriptors(md []metricDescriptor) markdownTable {
	marker := make(tableRow, columnCount)
	marker.populate(tableTopMarkerCenter)
	table := markdownTable{
		beginMarker: mdBeginMarker,
		header:      defaultHeader[:],
		topMarker:   marker,
		contents:    stringTable{},
		endMarker:   mdEndMarker,
	}
	for _, descriptor := range md {
		table.contents = append(table.contents, makeRow(descriptor))
	}
	return table
}

func loadConfig(configFile string) (models.Configuration, error) {
	var config models.Configuration

	input, err := readWholeFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal([]byte(input), &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return config, nil
}

func extractMetricDescriptors(configFile string) ([]metricDescriptor, error) {
	config, err := loadConfig(configFile)
	if err != nil {
		return nil, err
	}

	var descriptors []metricDescriptor //nolint:prealloc // Keep current configuration
	for _, collector := range config.Collectors {
		for _, metric := range collector.Metrics {
			metricName := strings.Join([]string{config.Namespace, collector.Name, metric.ID}, "_")
			varLabels := metric.DestLabels
			if len(metric.DestLabels) != len(metric.Labels) {
				varLabels = metric.Labels
			}
			descriptor := metricDescriptor{
				name:           metricName,
				metricType:     metric.Type,
				description:    metric.Help,
				constantLabels: metrics.ConstLabels[:],
				variableLabels: varLabels,
				query:          metric.Query,
			}
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors, nil
}

func extractLabelKeys(labelStr string) []string {
	var labelKeys []string
	pairs := strings.Split(labelStr, ",")
	for _, pair := range pairs {
		keyValue := strings.SplitN(pair, "=", 2)
		if len(keyValue) == 2 {
			labelKeys = append(labelKeys, keyValue[0])
		}
	}
	return labelKeys
}

func getVaultDescriptors() []metricDescriptor {
	metricName := fmt.Sprintf("%s_%s_%s", metrics.VaultMetricNamespace, metrics.VaultMonitorSubSystemName, metrics.VaultStatusSubSystemName)
	return []metricDescriptor{
		{
			name:           metricName,
			metricType:     vaultMetricType,
			description:    metrics.VaultStatusDescription,
			constantLabels: metrics.ConstLabels[:],
			variableLabels: []string{metrics.VaultInstanceLabelName},
			query:          "n/a (GET status)",
		},
	}
}

func extractDescriptors(spec metricSpecConfig, title string) ([]metricDescriptor, error) {
	// vault metrics are special case that needs to be handled separately
	// in future we may want to refactor the code to treat Vault as a generic collector
	if title == vaultMetricsTitle {
		descriptors := getVaultDescriptors()
		return descriptors, nil
	}

	var descriptors []metricDescriptor
	descriptors, err := extractMetricDescriptors(path.Join(rootPath, spec.metricConfigPath))
	if err != nil {
		return nil, err
	}

	return descriptors, nil
}

func generateDoc(spec metricSpecConfig, title string) (string, error) {
	var output strings.Builder

	descriptors, err := extractDescriptors(spec, title)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(&output, "\n## %s\n\n", title)
	table := buildTableFromDescriptors(descriptors)
	fmt.Fprintf(&output, "%v", table.toString())

	return output.String(), nil
}

func generateAllDocs(specs map[string]metricSpecConfig) (string, error) {
	var docs strings.Builder

	version, err := readVersion()
	if err != nil {
		return "", err
	}

	_, err = docs.WriteString(fmt.Sprintf("# Documentation of exported metrics in sre-exporter (v%s)\n", strings.TrimSpace(version)))
	if err != nil {
		return "", err
	}

	// sort the keys in order
	keys := make([]string, 0, len(specs))
	for key := range specs {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, title := range keys {
		doc, err := generateDoc(specs[title], title)
		if err != nil {
			return "", err
		}
		docs.WriteString(doc)
	}

	return docs.String(), nil
}
