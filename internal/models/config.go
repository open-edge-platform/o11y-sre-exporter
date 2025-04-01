// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	Name        string           `json:"name"`
	Enabled     bool             `json:"enabled"`
	Query       string           `json:"query"`
	ID          string           `json:"id"`
	Help        string           `json:"help"`
	Description *prometheus.Desc `json:"description"`
	Labels      []string         `json:"labels"`
	DestLabels  []string         `json:"destLabels"`
	Type        string           `json:"Type"`
}

type Collector struct {
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled"`
	Metrics []Metric `json:"metrics"`
}

type Source struct {
	URI string `json:"queryURI"`
	Org string `json:"mimirOrg"`
}

type Configuration struct {
	Namespace  string      `json:"namespace"`
	Source     Source      `json:"source"`
	Collectors []Collector `json:"collectors"`
}

type ConfigReloaderParameters struct {
	GRPCPort           string
	ConfigMapName      string
	ConfigName         string
	Namespace          string
	ReloadEndpoint     string
	ConfigHashEndpoint string
}
