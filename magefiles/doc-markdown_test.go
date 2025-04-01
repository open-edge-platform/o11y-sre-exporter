// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTableToMarkdown(t *testing.T) {
	//nolint:dupword // this is not important
	const expectedOutput = `<!-- Begin of auto-generated Markdown table -->
Name | Type | Description | Constant labels | Variable labels | Query
:---: | :---: | :---: | :---: | :---: | :---:
abc | abc | abc | abc | abc | abc
abc | abc | abc | abc | abc | abc
<!-- End of auto-generated Markdown table -->
`
	const columnCount = 6
	defaultHeader := [columnCount]string{"Name", "Type", "Description", "Constant labels", "Variable labels", "Query"}
	marker := make(tableRow, columnCount)
	marker.populate(tableTopMarkerCenter)
	table := markdownTable{
		beginMarker: mdBeginMarker,
		header:      defaultHeader[:],
		topMarker:   marker,
		contents:    stringTable{},
		endMarker:   mdEndMarker,
	}
	sampleRow := make(tableRow, columnCount)
	sampleRow.populate("abc")
	table.addRows(sampleRow)
	table.addRows(sampleRow)
	require.Equal(t, expectedOutput, table.toString())
}
