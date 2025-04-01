// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"crypto/sha256"
	"encoding/hex"
)

// getHash takes a byte slice and returns the SHA-256 hash as a string.
func getHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
