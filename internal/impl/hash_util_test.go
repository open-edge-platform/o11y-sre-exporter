// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"testing"
)

func TestGetHash(t *testing.T) {
	data := []byte("hello world")
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	hash := getHash(data)
	if hash != expectedHash {
		t.Errorf("expected %s, got %s", expectedHash, hash)
	}
}
