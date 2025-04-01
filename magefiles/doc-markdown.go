// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	"fmt"
	"strings"
)

type (
	tableRow    []string
	stringTable []tableRow
)

// markdownTable stores all what's needed to generate a markdown table
// it's up to user to validate Markdown correctness (e.g. equal column count).
type markdownTable struct {
	beginMarker string
	header      tableRow
	topMarker   tableRow
	contents    stringTable
	endMarker   string
}

func (row *tableRow) populate(elem string) {
	if row == nil {
		return
	}
	slice := *row
	for i := range slice {
		slice[i] = elem
	}
}

func (row *tableRow) toString() string {
	if row == nil {
		return ""
	}
	// escape '|' characters if exist
	for i, elem := range *row {
		(*row)[i] = strings.ReplaceAll(elem, "|", "\\|")
	}
	return strings.Join(*row, columnSeparator)
}

func (table *markdownTable) addRows(rows ...tableRow) {
	if table == nil {
		return
	}
	table.contents = append(table.contents, rows...)
}

func (table *markdownTable) toString() string {
	var output strings.Builder
	fmt.Fprintf(&output, "%v\n%v\n%v\n", table.beginMarker, table.header.toString(), table.topMarker.toString())
	for _, row := range table.contents {
		fmt.Fprintf(&output, "%v\n", row.toString())
	}
	fmt.Fprintln(&output, table.endMarker)
	return output.String()
}

const (
	mdBeginMarker        = "<!-- Begin of auto-generated Markdown table -->"
	mdEndMarker          = "<!-- End of auto-generated Markdown table -->"
	tableTopMarkerCenter = ":---:"
	columnSeparator      = " | "
)
