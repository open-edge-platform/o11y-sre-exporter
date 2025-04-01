// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

// InitConfig reads the configuration file and unmarshals it into a Configuration struct
// It returns hash of initial configuration and modifies it by normalizing the destination labels.
func InitConfig(configFile *string) (*models.Configuration, string, error) {
	var workingConfig = models.Configuration{}

	bytes, err := readConfig(configFile)
	if err != nil {
		return nil, "", fmt.Errorf("error reading configuration file: %w", err)
	}

	err = json.Unmarshal(bytes, &workingConfig)
	if err != nil {
		return nil, "", fmt.Errorf("unable to unmarshal: %w", err)
	}

	// get hash of original configuration
	hash, err := GetConfigHash(&workingConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to acquire hash: %w", err)
	}
	normalizeDestLabels(&workingConfig)
	log.Printf("InitConfig startup config %+v", workingConfig)
	return &workingConfig, hash, nil
}

func normalizeDestLabels(configuration *models.Configuration) {
	for i := 0; i < len(configuration.Collectors); i++ {
		collector := &configuration.Collectors[i]
		for j := 0; j < len(collector.Metrics); j++ {
			metric := &collector.Metrics[j]
			if len(metric.DestLabels) != len(metric.Labels) {
				metric.DestLabels = make([]string, len(metric.Labels))
				copy(metric.DestLabels, metric.Labels)
			}
		}
	}
}

func readConfig(configFile *string) ([]byte, error) {
	configBytes, err := os.ReadFile(*configFile)
	if err != nil {
		return configBytes, fmt.Errorf("error reading file %v: %w", *configFile, err)
	}
	return configBytes, nil
}

func GetConfigHash(config *models.Configuration) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("GetConfigHash: Failed to marshal config: %w", err)
	}
	return getHash(data), nil
}
