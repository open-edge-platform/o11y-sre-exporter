// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	pb "github.com/open-edge-platform/o11y-sre-exporter/api/config-reloader"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/impl"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

const (
	actionInitialize = "initialize"
	actionCleanup    = "cleanup"
	logFormat        = `Starting config-reloader with the following parameters:
------------------------------------------------------
gRPC Port: %s
ConfigMap Name: %s
Config Name: %s
Namespace: %s
Reload Endpoint: %s
Config Hash Endpoint: %s
------------------------------------------------------`
)

var (
	tenantIDNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9!_\-.*'()]+$`)
	grpcPort           = flag.String("grpcPort", "50051", "The gRPC config-reloader server port")
	configMapName      = flag.String("configMapName", "sre-exporter-config", "The name of the SRE ConfigMap")
	configName         = flag.String("configName", "sre-exporter-edge-node.json", "SRE exporter edge node config name")
	namespace          = flag.String("namespace", "orch-sre", "The SRE namespace")
	reloadEndpoint     = flag.String("reloadEndpoint", "http://localhost:9141/reload", "Metrics-exporter reload endpoint")
	configHashEndpoint = flag.String("configHashEndpoint", "http://localhost:9141/confighash", "Metrics-exporter config-hash endpoint")
)

type Server struct {
	pb.UnimplementedManagementServer

	gRPCPort           string
	configMapName      string
	configName         string
	namespace          string
	reloadEndpoint     string
	configHashEndpoint string
	podName            string

	clientset  kubernetes.Interface
	grpcServer *grpc.Server
	// mutex protects SRE Exporter configmap resource to ensure atomicity of tenant operations
	mutex sync.Mutex
}

// Starts the gRPC server.
func main() {
	flag.Parse()

	log.Printf(logFormat, *grpcPort, *configMapName, *configName, *namespace, *reloadEndpoint, *configHashEndpoint)

	configParams := models.ConfigReloaderParameters{
		GRPCPort:           *grpcPort,
		ConfigMapName:      *configMapName,
		ConfigName:         *configName,
		Namespace:          *namespace,
		ReloadEndpoint:     *reloadEndpoint,
		ConfigHashEndpoint: *configHashEndpoint,
	}

	server, err := NewServer(configParams)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+server.gRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	pb.RegisterManagementServer(server.grpcServer, server)

	log.Printf("Server listening on :%v", server.gRPCPort)

	// Register the health service
	healthCheck := health.NewServer()
	healthCheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(server.grpcServer, healthCheck)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		stop()

		log.Println("Got termination/interruption signal, attempting graceful shutdown.")
		stopped := make(chan struct{})
		go func() {
			server.grpcServer.GracefulStop()
			close(stopped)
		}()

		dur := 5 * time.Second
		t := time.NewTimer(dur)
		select {
		case <-t.C:
			log.Printf("Graceful shutdown could not be completed within %q, attempting ungraceful shutdown.", dur)
			server.grpcServer.Stop()
		case <-stopped:
			t.Stop()
		}

		wg.Done()
	}()

	log.Println("Starting grpc server.")
	if err := server.grpcServer.Serve(lis); err != nil {
		log.Panicf("Failed to serve: %v", err)
	}

	wg.Wait()
	log.Println("Shutdown completed.")
}

// NewServer creates a new Server instance with all necessary configurations and dependencies.
func NewServer(cfg models.ConfigReloaderParameters) (*Server, error) {
	// Initialize Kubernetes client
	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes incluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create new clientset: %w", err)
	}

	name := os.Getenv("POD_NAME")
	if name == "" {
		return nil, errors.New("failed to get pod name")
	}

	return &Server{
		gRPCPort:           cfg.GRPCPort,
		configMapName:      cfg.ConfigMapName,
		configName:         cfg.ConfigName,
		namespace:          cfg.Namespace,
		reloadEndpoint:     cfg.ReloadEndpoint,
		configHashEndpoint: cfg.ConfigHashEndpoint,
		clientset:          clientset,
		podName:            name,
		grpcServer:         grpc.NewServer(),
	}, nil
}

// InitializeTenant appends a new tenant ID from the request to the mimirOrg field
// in the relevant ConfigMap, then sends reload request to metrics-exporter.
func (s *Server) InitializeTenant(ctx context.Context, req *pb.TenantRequest) (*emptypb.Empty, error) {
	return s.processTenant(ctx, req, actionInitialize)
}

// CleanupTenant removes a tenant ID from the mimirOrg field in the relevant ConfigMap,
// then sends reload request to metrics-exporter.
func (s *Server) CleanupTenant(ctx context.Context, req *pb.TenantRequest) (*emptypb.Empty, error) {
	return s.processTenant(ctx, req, actionCleanup)
}

// processTenant handles the common logic for initializing and cleaning up tenants.
func (s *Server) processTenant(ctx context.Context, req *pb.TenantRequest, action string) (*emptypb.Empty, error) {
	tenant := req.GetTenant()
	log.Printf("Received %s tenant request: %q", action, tenant)
	defer log.Printf("Tenant %s request handling completed.", action)

	err := validateTenantID(tenant)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant name: %v", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	configMap, err := getConfigMap(ctx, s.namespace, s.configMapName, s.clientset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get ConfigMap: %v", err)
	}

	var configData models.Configuration
	var respCode codes.Code

	switch action {
	case actionInitialize:
		configData, respCode, err = addTenant(configMap, s.configName, tenant)
		// we don't treat AlreadyExists as an error, as it means the tenant is already initialized
		if err != nil && respCode != codes.AlreadyExists {
			return nil, status.Errorf(respCode, "failed to initialize tenant: %v", err)
		}
	case actionCleanup:
		configData, respCode, err = removeTenant(configMap, s.configName, tenant)
		// we don't treat NotFound as an error, as it means the tenant is already cleaned up
		if err != nil && respCode != codes.NotFound {
			return nil, status.Errorf(respCode, "failed to cleanup tenant: %v", err)
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown action: %v", action)
	}

	// Skip updating ConfigMap if tenant doesn't exist there, but reload metrics exporter anyway
	if (action == actionInitialize && respCode != codes.AlreadyExists) || (action == actionCleanup && respCode != codes.NotFound) {
		if err := updateConfigMap(ctx, s.clientset, configData, s.configMapName, s.configName, s.namespace); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update ConfigMap: %v", err)
		}
	}

	configMapHash, err := impl.GetConfigHash(&configData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to acquire hash from configmap: %v", err)
	}

	containerConfigHash, err := getConfigHashFromContainer(ctx, s.configHashEndpoint)
	if err != nil {
		log.Printf("Failed to acquire hash from endpoint %s: %v", s.configHashEndpoint, err)
		return nil, status.Errorf(codes.Unavailable, "Failed to acquire hash from container: %v", err)
	}

	if configMapHash != containerConfigHash {
		pod, err := getPod(ctx, s.namespace, s.podName, s.clientset)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get Pod: %v", err)
		}

		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations["configMapHash"] = configMapHash

		if err := updatePod(ctx, s.clientset, pod, s.podName, s.namespace); err != nil {
			log.Printf("Failed to update Pod annotation: %v", err)
		}

		log.Printf("ConfigMap hash %s and container config hash %s do not match, reloading metrics-exporter", configMapHash, containerConfigHash)
		if err := sendReloadRequestToContainer(ctx, s.reloadEndpoint); err != nil {
			return nil, status.Errorf(codes.Unavailable, "failed to reload metrics-exporter: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Container metrics-exporter reloaded because it did not use the latest config")
	}

	log.Printf("The container's config hash matches the hash of updated ConfigMap. Tenant action %s completed successfully.", action)

	return &emptypb.Empty{}, nil
}

// getConfigMap retrieves the ConfigMap with the specified name from the given namespace.
func getConfigMap(ctx context.Context, namespace, configMapName string, clientset kubernetes.Interface) (*corev1.ConfigMap, error) {
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	return configMap, nil
}

// getPod retrieves the Pod with the specified name from the given namespace.
func getPod(ctx context.Context, namespace, podName string, clientset kubernetes.Interface) (*corev1.Pod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Pod: %w", err)
	}

	return pod, nil
}

// validateTenantID validates the tenant ID based on Mimir restrictions.
// https://grafana.com/docs/mimir/latest/configure/about-tenant-ids/
func validateTenantID(tenantID string) error {
	// Check if tenant ID is empty.
	if tenantID == "" {
		return errors.New("tenant ID cannot be empty")
	}

	// Tenant IDs must be <= 150 bytes or characters in length.
	if len(tenantID) > 150 {
		return errors.New("tenant ID exceeds 150 characters")
	}

	// Forbidden tenant ID patterns.
	if tenantID == "." || tenantID == ".." || tenantID == "__mimir_cluster" {
		return errors.New("tenant ID cannot be '.' or '..' or '__mimir_cluster'")
	}

	// Only allowed characters: Alphanumeric + special characters defined.
	matches := tenantIDNameRegexp.MatchString(tenantID)
	if !matches {
		return errors.New("tenant ID contains unsupported characters")
	}

	return nil
}

// updateConfigMap updates an existing ConfigMap in the Kubernetes cluster
// with the provided configuration data.
func updateConfigMap(ctx context.Context, clientset kubernetes.Interface, configData models.Configuration, configMapName, configName, namespace string) error {
	patchedData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal patched data: %w", err)
	}

	patchStruct := struct {
		Data map[string]string `json:"data"`
	}{
		Data: map[string]string{
			configName: string(patchedData),
		},
	}

	patchBytes, err := json.Marshal(patchStruct)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Patch(ctx, configMapName, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch ConfigMap: %w", err)
	}

	return nil
}

// updatePod updates an existing Pod in the Kubernetes cluster
// with the provided configuration data.
func updatePod(ctx context.Context, clientset kubernetes.Interface, patchedPod *corev1.Pod, podName, namespace string) error {
	patchBytes, err := json.Marshal(patchedPod)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	_, err = clientset.CoreV1().Pods(namespace).Patch(ctx, podName, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch pod: %w", err)
	}

	return nil
}

// sendReloadRequestToContainer sends a POST request to the localhost address of the
// metric-exporter container to trigger a configuration reload.
func sendReloadRequestToContainer(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send request, status code: %d", resp.StatusCode)
	}

	return nil
}

// getConfigHashFromContainer sends a GET request to the given endpoint
// and returns the response body as a string on success.
func getConfigHashFromContainer(ctx context.Context, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// addTenant adds a new tenant to the mimirOrg field in the configuration if it doesn't already exist.
func addTenant(configMap *corev1.ConfigMap, configName, tenant string) (models.Configuration, codes.Code, error) {
	jsonData := configMap.Data[configName]

	var configData models.Configuration
	err := json.Unmarshal([]byte(jsonData), &configData)
	if err != nil {
		return models.Configuration{}, codes.Internal, fmt.Errorf("failed to unmarshal ConfigMap data: %w", err)
	}

	tenantList := strings.Split(configData.Source.Org, "|")
	if slices.Contains(tenantList, tenant) {
		return configData, codes.AlreadyExists, fmt.Errorf("tenant %q already exists", tenant)
	}

	tenantList = append(tenantList, tenant)
	configData.Source.Org = strings.Join(tenantList, "|")

	return configData, codes.OK, nil
}

// removeTenant removes an existing tenant from the mimirOrg field in the configuration.
func removeTenant(configMap *corev1.ConfigMap, configName, tenant string) (models.Configuration, codes.Code, error) {
	jsonData := configMap.Data[configName]

	var configData models.Configuration
	err := json.Unmarshal([]byte(jsonData), &configData)
	if err != nil {
		return models.Configuration{}, codes.Internal, fmt.Errorf("failed to unmarshal ConfigMap data: %w", err)
	}

	tenantList := strings.Split(configData.Source.Org, "|")
	updatedTenants := slices.DeleteFunc(tenantList, func(t string) bool {
		return t == tenant
	})

	if len(updatedTenants) == len(tenantList) {
		return configData, codes.NotFound, fmt.Errorf("tenant %q not found", tenant)
	}

	configData.Source.Org = strings.Join(updatedTenants, "|")

	return configData, codes.OK, nil
}
