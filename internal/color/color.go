// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package color

type Color string

const (
	Info  Color = "\033[36m"
	Good  Color = "\033[32m"
	Warn  Color = "\033[33m"
	Error Color = "\033[31m"

	reset Color = "\033[0m"
)

func FormatString(color Color, msg string) string {
	if len(msg) == 0 {
		return ""
	}

	return string(color) + msg + string(reset)
}
