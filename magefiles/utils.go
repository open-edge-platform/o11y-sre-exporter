// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/sh"
)

func getCurrentBranch() (string, error) {
	cmd := []string{"git", "branch", "--show-current"}
	out, err := sh.Output(cmd[0], cmd[1:]...)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch in sre-exporter repository: %w", err)
	}
	return out, nil
}

func readWholeFile(filePath string) (string, error) {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return string(contents), nil
}

func readVersion() (string, error) {
	version, err := readWholeFile(versionFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read version: %w", err)
	}
	return version, nil
}
