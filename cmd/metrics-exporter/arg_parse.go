// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"strings"
)

type argList []string

// Interface needed to implement to allow multiple vaultURI entries.
func (URIs *argList) String() string {
	return strings.Join(*URIs, ", ")
}
func (URIs *argList) Set(value string) error {
	*URIs = append(*URIs, value)
	return nil
}

var (
	vaultURIs      argList
	configFiles    argList
	listenAddress  = flag.String("listenAddress", ":9141", "local <address>:port for sre-exporter to listen on")
	customerLabel  = flag.String("customerLabel", "UNKNOWN_CUSTOMER", "Value of the customer label to use in the exported metrics")
	vaultNamespace = flag.String("vaultNamespace", "orch-platform", "K8S namespace where vault pods are running")
	ver            = flag.Bool("version", false, "prints current version")

	startUpFmt = `
Metrics-Exporter v%s starting up with the following parameters:
	vaultURIs: %s
	configFiles: %s
	listenAddress: %s
	customerLabel: %s
	vaultNamespace: %s`
)

func parseArgs() {
	flag.Var(&configFiles, "config", "filename of json file that holds collector and metric data")
	flag.Var(&vaultURIs, "vaultURI", "URI to contact vault via NOTE this can be set multiple times")
	flag.Parse()
}
