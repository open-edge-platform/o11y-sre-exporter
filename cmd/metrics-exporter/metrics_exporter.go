// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	k8s "k8s.io/client-go/kubernetes"
	k8s_rest "k8s.io/client-go/rest"

	"github.com/open-edge-platform/o11y-sre-exporter/internal/color"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/impl"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/metrics"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/scraping"
)

const (
	tickInterval        = 5 * time.Minute
	otelMetricsEndpoint = "http://127.0.0.1:8888/metrics"
)

var (
	version string
)

func main() {
	parseArgs()

	if *ver {
		fmt.Println("SRE-Exporter version: ", version)
		os.Exit(0)
	}

	startUpMessage := fmt.Sprintf(startUpFmt, version, vaultURIs, configFiles, *listenAddress, *customerLabel, *vaultNamespace)
	log.Println(color.FormatString(color.Info, startUpMessage))

	// Run scraping goroutine which scrapes OpenTelemetry Collector endpoint
	// this goroutine runs in the background and does not block the main server goroutine
	runScrapingManager()
	pipelineManager := impl.NewPipelineManager(listenAddress)
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// the loop running main server goroutine
	// restarts on SIGHUP signal sent to reload the configuration
	serverStarted := false
	for {
		err := initializePipelineManager(pipelineManager, configFiles, vaultURIs, vaultNamespace, customerLabel, done)
		if err != nil {
			log.Fatalf("Failed to initialize pipeline manager: %v", err)
		}

		// start the server only once
		if !serverStarted {
			go func() {
				log.Print("Serving metrics")
				if err := pipelineManager.Start(); !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("Server error: %v", err)
				}
			}()
			serverStarted = true
		}

		// Blocks here
		sig := <-done
		// signal other than SIGHUP received, shutdown the server
		if sig != syscall.SIGHUP {
			log.Printf("Received signal: %v", sig)
			break
		}
	}
	log.Print("Server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pipelineManager.Shutdown(ctx); err != nil {
		log.Printf("Graceful server shutdown failed: %v", err)
		if err := pipelineManager.Close(); err != nil {
			log.Panicf("Could not properly close the server: %v", err)
		}
	} else {
		log.Print("Server shut down properly")
	}
}

// initializePipelineManager initializes the pipeline manager and returns it along with any error encountered.
func initializePipelineManager(pipelineManager *impl.PipelineManager, configFiles []string, vaultURIs []string,
	vaultNamespace, customerLabel *string, done chan os.Signal) error {
	pipelines := make([]*impl.Pipeline, 0)
	configHash := make(map[string]string)

	for i := range configFiles {
		config, hash, err := impl.InitConfig(&configFiles[i])
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		configHash[config.Namespace] = hash

		collectors, err := metrics.BuildCollectorsFromConfig(config, *customerLabel)
		if err != nil {
			return fmt.Errorf("failed to build collectors: %w", err)
		}
		pipeline := impl.NewPipeline(config.Namespace)
		pipeline.AddCollectors(collectors...)
		pipelines = append(pipelines, pipeline)
	}

	vaultCollector, err := newVaultCollector(vaultURIs, vaultNamespace, customerLabel)
	if err != nil {
		return fmt.Errorf("failed to initialize Vault collector: %w", err)
	}
	pipeline := impl.NewPipeline("vault")
	pipeline.AddCollectors(vaultCollector)
	pipelines = append(pipelines, pipeline)

	for i := range pipelines {
		pipelineManager.RegisterPipeline("/"+pipelines[i].GetNamespace()+"/metrics", pipelines[i])
	}

	// Register health check
	pipelineManager.RegisterHealthCheck()
	// Register the reload endpoint
	pipelineManager.RegisterReload("/reload", done)
	// Register the config hash endpoint with edgenode configmap hash
	pipelineManager.RegisterConfigHash("/confighash", configHash["orch_edgenode"])

	return nil
}

// newVaultCollector initializes the Kubernetes client and returns a VaultSynthCollector.
func newVaultCollector(vaultURIs []string, vaultNamespace, customerLabel *string) (prometheus.Collector, error) {
	config, err := k8s_rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize k8s client config: %w", err)
	}
	k8sCli, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize k8s client: %w", err)
	}
	return metrics.NewVaultSynthCollector(k8sCli, vaultURIs, *vaultNamespace, metrics.DefaultPodPort, *customerLabel), nil
}

func runScrapingManager() {
	// Start the scraping manager
	scrapingManager := scraping.NewScrapeEventDesc(otelMetricsEndpoint, scrapedOtelMetricsList)
	ticker := time.NewTicker(tickInterval)
	// This goroutine will never exit.
	// TODO: implement proper way of handling ticker in goroutine as e.g. in the following code
	// https://github.com/open-edge-platform/o11y-alerting-monitor/blob/main/internal/executor/executor.go#L63-L99
	go func() {
		for range ticker.C {
			err := scrapingManager.ScrapeMetrics()
			if err != nil {
				log.Printf("Failed to scrape metrics: %v", err)
			}
		}
	}()
}
