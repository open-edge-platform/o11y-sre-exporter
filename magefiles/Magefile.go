// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/magefile/mage/mg"
)

const (
	sourcesPath    = "path: deployments/sre-exporter"
	sourcesRepoURL = "repoURL: https://github.com/open-edge-platform/o11y-sre-exporter"
	templatePath   = "../edge-manageability-framework/argocd/applications/templates/sre-exporter.yaml"
)

var (
	repoPattern   = regexp.MustCompile(`(?m:repoURL:.*$)`)
	pathPattern   = regexp.MustCompile(`(?m:chart:.*$|path:.*$)`)
	targetPattern = regexp.MustCompile(`(?m:targetRevision:.*$)`)
)

type Argo mg.Namespace

// UpdateSreTemplate updates sre-exporter template in edge-manageability-framework with current branch.
func (Argo) UpdateSreTemplate() error {
	input, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read sre-exporter template: %w", err)
	}

	currBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	oRepo := repoPattern.ReplaceAllLiteralString(string(input), sourcesRepoURL)
	oPath := pathPattern.ReplaceAllLiteralString(oRepo, sourcesPath)
	oTarget := targetPattern.ReplaceAllLiteralString(oPath, "targetRevision: "+currBranch)

	if err := os.WriteFile(templatePath, []byte(oTarget), 0640); err != nil {
		return fmt.Errorf("failed to write modified sre-exporter template: %w", err)
	}

	fmt.Printf("Successfully updated sre-exporter template in edge-manageability-framework with branch %s\n", currBranch)
	return nil
}

type Doc mg.Namespace

// Generate generates documentation of exported metrics.
func (Doc) Generate() error {
	docs, err := generateAllDocs(metricSpecs)
	if err != nil {
		return err
	}
	docFile, err := os.Create(docPath)
	if err != nil {
		return err
	}
	defer docFile.Close()

	_, err = docFile.WriteString(docs)
	if err != nil {
		return err
	}
	return nil
}
