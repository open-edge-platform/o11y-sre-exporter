// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package impl

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

type routerSwapper struct {
	mu     sync.Mutex
	router *mux.Router
}

func (rs *routerSwapper) Swap(newRouter *mux.Router) {
	rs.mu.Lock()
	rs.router = newRouter
	rs.mu.Unlock()
}

func (rs *routerSwapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	router := rs.router
	rs.mu.Unlock()
	router.ServeHTTP(w, r)
}

type PipelineManager struct {
	routerSwapper *routerSwapper
	server        *http.Server
	pipelines     []*Pipeline
}

func NewPipelineManager(listenAddress *string) *PipelineManager {
	swapper := &routerSwapper{}
	swapper.Swap(mux.NewRouter())

	server := &http.Server{
		Addr:         *listenAddress,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		Handler:      swapper,
	}

	return &PipelineManager{
		routerSwapper: swapper,
		server:        server,
	}
}

func (manager *PipelineManager) RegisterPipeline(endpoint string, pipeline *Pipeline) {
	manager.routerSwapper.router.Handle(endpoint, pipeline.GetEndpointHandler())
	log.Printf("endpoint %q registered", endpoint)
	manager.pipelines = append(manager.pipelines, pipeline)
}

func (manager *PipelineManager) RegisterReload(endpoint string, done chan os.Signal) {
	manager.routerSwapper.router.HandleFunc(endpoint, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			log.Printf("Received %v request on %v endpoint", req.Method, endpoint)
			http.Error(w, "Bad request", http.StatusMethodNotAllowed)
			return
		}

		log.Print("Received hot reload request. Reinitializing pipeline manager...")
		if err := manager.CleanUp(); err != nil {
			log.Printf("Failed to clean up pipeline manager: %v", err)
			http.Error(w, "Request failed", http.StatusInternalServerError)
			// hot reload impossible, so we need to restart the process
			done <- syscall.SIGTERM
			return
		}
		w.WriteHeader(http.StatusOK)
		done <- syscall.SIGHUP
	})
	log.Printf("endpoint %q registered", endpoint)
}

func (manager *PipelineManager) RegisterConfigHash(endpoint string, hash string) {
	manager.routerSwapper.router.HandleFunc(endpoint, func(w http.ResponseWriter, req *http.Request) {
		log.Printf("Received %v request on %q endpoint", req.Method, endpoint)
		if req.Method != http.MethodGet {
			http.Error(w, "Bad request", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("Responding with hash: %s", hash)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(hash))
		if err != nil {
			log.Print(err.Error())
		}
	})
	log.Printf("endpoint %q registered", endpoint)
}

// RegisterHealthCheck registers the health check endpoint.
func (manager *PipelineManager) RegisterHealthCheck() {
	manager.routerSwapper.router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Print(err.Error())
		}
	})
	log.Print("Health check endpoint registered")
}

func (manager *PipelineManager) Start() error {
	return manager.server.ListenAndServe()
}

func (manager *PipelineManager) Shutdown(ctx context.Context) error {
	return manager.server.Shutdown(ctx)
}

func (manager *PipelineManager) Close() error {
	return manager.server.Close()
}

// CleanUp unregisters collectors for each pipeline, removes all pipelines, and swaps a new router.
func (manager *PipelineManager) CleanUp() error {
	for _, pipeline := range manager.pipelines {
		err := pipeline.UnregisterCollectors()
		if err != nil {
			return fmt.Errorf("could not clear pipeline '%v', error: %w", pipeline, err)
		}
	}
	manager.routerSwapper.Swap(mux.NewRouter())
	manager.pipelines = nil

	return nil
}
