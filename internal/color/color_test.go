// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package color

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatString(t *testing.T) {
	msg := "tes3t# meSsage!"
	var tests = []struct {
		name     string
		color    Color
		msg      string
		expected string
	}{
		{"EmptyMsg", "", "", ""},
		{"EmptyColor", "", msg, msg + "\033[0m"},
		{"Info", Info, msg, fmt.Sprintf("\033[36m%s\033[0m", msg)},
		{"Good", Good, msg, fmt.Sprintf("\033[32m%s\033[0m", msg)},
		{"Warn", Warn, msg, fmt.Sprintf("\033[33m%s\033[0m", msg)},
		{"Error", Error, msg, fmt.Sprintf("\033[31m%s\033[0m", msg)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatString(tt.color, tt.msg)
			require.Equal(t, tt.expected, actual)
		})
	}
}
