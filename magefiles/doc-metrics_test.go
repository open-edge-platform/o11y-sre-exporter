// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsPresence(t *testing.T) {
	// required only for tests in /magefiles folder
	// mage by default sets workdir to root of repository
	// but go test has /magefiles as workdir
	wd, err := os.Getwd()
	require.NoError(t, err)
	rootPath = path.Join(wd, "..")

	for title, spec := range metricSpecs {
		descriptors, err := extractDescriptors(spec, title)
		require.NoError(t, err)

		metricsBaseline, err := readWholeFile(path.Join(rootPath, spec.metricBaseline))
		require.NoError(t, err)

		for _, desc := range descriptors {
			if !slices.Contains(spec.doNotValidateMetrics, desc.name) {
				testMetric(t, desc, metricsBaseline)
			}
		}
	}
}

func testMetric(t *testing.T, desc metricDescriptor, output string) {
	// test metric type
	typeStr := fmt.Sprintf("# TYPE %s %s", desc.name, strings.ToLower(desc.metricType))
	require.Containsf(t, output, typeStr,
		"The reference metrics from UT do not contain the expected substring: \n%s\n\n==== Reference metrics below ====\n%s\n",
		typeStr, output)
	// test metric help
	helpStr := fmt.Sprintf("# HELP %s %s", desc.name, desc.description)
	require.Containsf(t, output, helpStr,
		"The reference metrics from UT do not contain the expected substring: \n%s\n\n==== Reference metrics below ====\n%s\n",
		helpStr, output)
	// test metric itself
	found := false
	expectedLabels := append(desc.constantLabels, desc.variableLabels...)
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "# HELP") || strings.HasPrefix(line, "# TYPE") {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			if desc.name == matches[1] {
				found = true
				labelsAndValues := extractLabelKeys(matches[2])
				require.ElementsMatch(t, expectedLabels, labelsAndValues, "existing labels do not match expected labels")
			}
		}
	}
	require.Truef(t, found, "Metric %s not found in the output!", desc.name)
}

func TestGenerateDoc(t *testing.T) {
	title := "1. Fake Exported Metrics"
	metricConfigPath := "magefiles/testdata/fakeConfig.json"
	metricBaseline := "testdata/fakeExpectedOutput.txt"

	spec := metricSpecConfig{
		metricConfigPath:     metricConfigPath,
		metricBaseline:       metricBaseline,
		doNotValidateMetrics: []string{},
	}

	expectedDoc, err := os.ReadFile(metricBaseline)
	if err != nil {
		t.Fatalf("failed to read expected output: %v", err)
	}

	doc, err := generateDoc(spec, title)
	require.NoError(t, err)

	require.Equal(t, string(expectedDoc), doc)
}
