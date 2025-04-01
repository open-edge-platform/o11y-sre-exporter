// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package models

import "fmt"

type ComparisionType string

const (
	RateThreshold  ComparisionType = "RateThreshold"
	ValueThreshold ComparisionType = "ValueThreshold"
)

// TODO add in future also []{key, value string} to the metric.
type ScrapeEventCondition struct {
	Metric     string
	CheckType  ComparisionType
	Threshold  float64
	MetricType string
}

func (se ScrapeEventCondition) ToString() string {
	return fmt.Sprintf("ScrapeEventCondition - Metric: %v CheckType: %v Threshold: %v MetricType: %v",
		se.Metric, se.CheckType, se.Threshold, se.MetricType)
}
