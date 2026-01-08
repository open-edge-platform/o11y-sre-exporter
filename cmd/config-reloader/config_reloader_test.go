// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	pb "github.com/open-edge-platform/o11y-sre-exporter/api/config-reloader"
	"github.com/open-edge-platform/o11y-sre-exporter/internal/models"
)

const (
	sreConfigMapName = "sre-exporter-config"
	sreNamespace     = "orch-sre" // mock K8S namespace for sre resources.
	sreConfigName    = "sre-exporter-edge-node.json"
	srePodName       = "sre-exporter-7cf4cc4656-tbls6"
)

var initialConfig = `
{
	"namespace": "orch_edgenode",
	"source": {
		"queryURI": "http://testurl:8181/prometheus",
		"mimirOrg": "tenant1|tenant2"
	},
	"collectors": null
}`

func TestGetConfigMap(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		configMapName     string
		expectedConfigMap *corev1.ConfigMap
		expectError       bool
	}{
		{
			name:          "successful configmap retrieval",
			namespace:     sreNamespace,
			configMapName: "test-configmap",
			expectedConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test-configmap", Namespace: sreNamespace},
				Data: map[string]string{
					"test-config.json": initialConfig,
				},
			},
			expectError: false,
		},
		{
			name:              "failed configmap retrieval",
			namespace:         sreNamespace,
			configMapName:     "non-existent",
			expectedConfigMap: nil,
			expectError:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewClientset()
			if test.expectedConfigMap != nil {
				_, err := clientset.CoreV1().ConfigMaps(test.namespace).Create(t.Context(), test.expectedConfigMap, metav1.CreateOptions{})
				require.NoError(t, err, "Failed to create ConfigMap in fake clientset")
			}
			if test.expectError {
				_, err := getConfigMap(t.Context(), test.namespace, test.configMapName, clientset)
				require.Error(t, err)
			} else {
				cm, err := getConfigMap(t.Context(), test.namespace, test.configMapName, clientset)
				require.NoError(t, err)
				require.Equal(t, test.expectedConfigMap.Name, cm.Name)
			}
		})
	}
}

func TestGetPod(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		podName     string
		expectedPod *corev1.Pod
		expectError bool
	}{
		{
			name:      "successful pod retrieval",
			namespace: sreNamespace,
			podName:   "test-pod",
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: sreNamespace},
			},
			expectError: false,
		},
		{
			name:        "failed pod retrieval",
			namespace:   sreNamespace,
			podName:     "non-existent",
			expectedPod: nil,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewClientset()
			if test.expectedPod != nil {
				_, err := clientset.CoreV1().Pods(test.namespace).Create(t.Context(), test.expectedPod, metav1.CreateOptions{})
				require.NoError(t, err, "Failed to create ConfigMap in fake clientset")
			}
			pod, err := getPod(t.Context(), test.namespace, test.podName, clientset)
			if test.expectError {
				require.Nil(t, pod)
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedPod.Name, pod.Name)
			}
		})
	}
}

func TestValidateTenantID(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Tenant ID",
			tenantID:    "tenant1",
			expectError: false,
		},
		{
			name:        "Empty Tenant ID",
			tenantID:    "",
			expectError: true,
			errorMsg:    "tenant ID cannot be empty",
		},
		{
			name:        "Tenant ID Too Long",
			tenantID:    string(make([]byte, 151)), // 151 characters
			expectError: true,
			errorMsg:    "tenant ID exceeds 150 characters",
		},
		{
			name:        "Forbidden Tenant ID '.'",
			tenantID:    ".",
			expectError: true,
			errorMsg:    "tenant ID cannot be '.' or '..' or '__mimir_cluster'",
		},
		{
			name:        "Forbidden Tenant ID '__mimir_cluster'",
			tenantID:    "__mimir_cluster",
			expectError: true,
			errorMsg:    "tenant ID cannot be '.' or '..' or '__mimir_cluster'",
		},
		{
			name:        "Tenant ID with invalid characters",
			tenantID:    "invalid_id@",
			expectError: true,
			errorMsg:    "tenant ID contains unsupported characters",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTenantID(test.tenantID)
			if test.expectError {
				require.Error(t, err, "Expected an error for tenant ID: %s", test.tenantID)
				require.Contains(t, err.Error(), test.errorMsg, "Error message mismatch")
			} else {
				require.NoError(t, err, "Did not expect an error for tenant ID: %s", test.tenantID)
			}
		})
	}
}

func TestSendReloadRequestToContainer(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		endpoint    string
		expectError bool
	}{
		{
			name: "Successful Request",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			endpoint:    "/reload",
			expectError: false,
		},
		{
			name: "Failed Request - 500 Internal Server Error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			endpoint:    "/reload",
			expectError: true,
		},
		{
			name:        "Invalid URL",
			handler:     nil,
			endpoint:    "://invalid-url",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var server *httptest.Server
			if test.handler != nil {
				server = httptest.NewServer(test.handler)
				defer server.Close()
				test.endpoint = server.URL + test.endpoint
			}

			err := sendReloadRequestToContainer(t.Context(), test.endpoint)
			if test.expectError {
				require.Error(t, err, "Expected an error for endpoint: %q", test.endpoint)
			} else {
				require.NoError(t, err, "Did not expect an error for endpoint: %q", test.endpoint)
			}
		})
	}
}

func TestUpdateConfigMap(t *testing.T) {
	updatedConfig := models.Configuration{
		Namespace: "orch_edgenode",
		Source: models.Source{
			URI: "http://testurl:8181/prometheus",
			Org: "tenant3",
		},
	}

	tests := []struct {
		name          string
		updatedConfig models.Configuration
		expectError   bool
	}{
		{
			name:          "successful update",
			updatedConfig: updatedConfig,
			expectError:   false,
		},
	} // consider adding negative case

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-configmap",
						Namespace: sreNamespace,
					},
					Data: map[string]string{
						"test-config.json": initialConfig,
					},
				},
			)
			err := updateConfigMap(t.Context(), clientset, test.updatedConfig, "test-configmap", "test-config.json", sreNamespace)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				configMapAfterUpdate, err := clientset.CoreV1().ConfigMaps(sreNamespace).Get(t.Context(), "test-configmap", metav1.GetOptions{})
				require.NoError(t, err)

				updatedData := configMapAfterUpdate.Data["test-config.json"]

				var updatedConfiguration models.Configuration
				err = json.Unmarshal([]byte(updatedData), &updatedConfiguration)

				require.NoError(t, err)
				require.Equal(t, updatedConfig, updatedConfiguration)
			}
		})
	}
}

func TestUpdatePod(t *testing.T) {
	annotations := make(map[string]string)
	annotations["test"] = "test"

	tests := []struct {
		name        string
		updatedPod  *corev1.Pod
		expectError bool
	}{
		{
			name: "successful update",
			updatedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: sreNamespace, Annotations: annotations},
			},
			expectError: false,
		},
	} // consider adding negative case

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewClientset(
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: sreNamespace,
					},
				},
			)
			err := updatePod(t.Context(), clientset, test.updatedPod, "test-pod", sreNamespace)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				podAfterUpdate, err := clientset.CoreV1().Pods(sreNamespace).Get(t.Context(), "test-pod", metav1.GetOptions{})
				require.NoError(t, err)

				require.NoError(t, err)
				require.Equal(t, test.updatedPod, podAfterUpdate)
			}
		})
	}
}

func TestAddTenant(t *testing.T) {
	tests := []struct {
		name          string
		configMap     *corev1.ConfigMap
		configName    string
		tenant        string
		expectedError bool
		expectedOrg   string
		expectedCode  codes.Code
	}{
		{
			name: "successfully add tenant",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"test-config.json": initialConfig,
				},
			},
			configName:    "test-config.json",
			tenant:        "tenant3",
			expectedError: false,
			expectedOrg:   "tenant1|tenant2|tenant3",
			expectedCode:  codes.OK,
		},
		{
			name: "tenant already exists",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"test-config.json": initialConfig,
				},
			},
			configName:    "test-config.json",
			tenant:        "tenant1",
			expectedError: true,
			expectedOrg:   "tenant1|tenant2",
			expectedCode:  codes.AlreadyExists,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, respCode, err := addTenant(test.configMap, test.configName, test.tenant)
			if test.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedOrg, config.Source.Org)
			}
			require.Equal(t, test.expectedCode, respCode)
		})
	}
}

func TestRemoveTenant(t *testing.T) {
	tests := []struct {
		name          string
		configMap     *corev1.ConfigMap
		configName    string
		tenant        string
		expectedError bool
		expectedOrg   string
		expectedCode  codes.Code
	}{
		{
			name: "successfully remove tenant",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"test-config.json": initialConfig,
				},
			},
			configName:    "test-config.json",
			tenant:        "tenant2",
			expectedError: false,
			expectedOrg:   "tenant1",
			expectedCode:  codes.OK,
		},
		{
			name: "tenant not found",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"test-config.json": initialConfig,
				},
			},
			configName:    "test-config.json",
			tenant:        "tenant3",
			expectedError: true,
			expectedOrg:   "tenant1|tenant2",
			expectedCode:  codes.NotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, respCode, err := removeTenant(test.configMap, test.configName, test.tenant)
			if test.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedOrg, config.Source.Org)
			}
			require.Equal(t, test.expectedCode, respCode)
		})
	}
}

func TestProcessTenant(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reload":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("success"))
			require.NoError(t, err, "Failed to write response")
		case "/confighash":
			w.WriteHeader(http.StatusOK)
			// return hash of initialConfig. Instruction for update:
			// 1. run `go test -v -run TestProcessTenant ./...`
			// 2. look for the following string in the failing test case:
			// "ConfigMap hash <...> and container config hash <...> do not match"
			// 3. replace the hash below with the 1st hash from the error message
			_, err := w.Write([]byte("992ea0311294f8aeef0e0c0720a5d00fac66c6e4dbd615679d00ad9e5a4f2681"))
			require.NoError(t, err, "Failed to write response")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tests := []struct {
		name              string
		action            string
		tenantRequest     *pb.TenantRequest
		configMap         *corev1.ConfigMap
		pod               *corev1.Pod
		expectedPodUpdate bool
		expectedError     bool
		expectedCode      codes.Code
		expectedConfig    string
	}{
		{
			name:   "initialize tenant",
			action: actionInitialize,
			tenantRequest: &pb.TenantRequest{
				Tenant: "tenant3",
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sreConfigMapName,
					Namespace: sreNamespace,
				},
				Data: map[string]string{
					sreConfigName: initialConfig,
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      srePodName,
					Namespace: sreNamespace,
				},
			},
			expectedPodUpdate: true,
			expectedError:     true,
			expectedCode:      codes.Internal,
			expectedConfig: `{
				"namespace":"orch_edgenode",
				"source":{
					"queryURI":"http://testurl:8181/prometheus",
					"mimirOrg":"tenant1|tenant2|tenant3"
				},
				"collectors": null
			}`,
		},
		{
			name:   "initialize existing tenant",
			action: actionInitialize,
			tenantRequest: &pb.TenantRequest{
				Tenant: "tenant1",
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sreConfigMapName,
					Namespace: sreNamespace,
				},
				Data: map[string]string{
					sreConfigName: initialConfig,
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      srePodName,
					Namespace: sreNamespace,
				},
			},
			expectedPodUpdate: false,
			expectedError:     false,
			expectedCode:      codes.OK,
			expectedConfig: `{
				"namespace":"orch_edgenode",
				"source":{
					"queryURI":"http://testurl:8181/prometheus",
					"mimirOrg":"tenant1|tenant2"
				},
				"collectors": null
			}`,
		},
		{
			name:   "cleanup tenant",
			action: actionCleanup,
			tenantRequest: &pb.TenantRequest{
				Tenant: "tenant2",
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sreConfigMapName,
					Namespace: sreNamespace,
				},
				Data: map[string]string{
					sreConfigName: initialConfig,
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      srePodName,
					Namespace: sreNamespace,
				},
			},
			expectedPodUpdate: true,
			expectedError:     true,
			expectedCode:      codes.Internal,
			expectedConfig: `{
				"namespace":"orch_edgenode",
				"source":{
					"queryURI":"http://testurl:8181/prometheus",
					"mimirOrg":"tenant1"
				},
				"collectors": null
			}`,
		},
		{
			name:   "cleanup non-existing tenant",
			action: actionCleanup,
			tenantRequest: &pb.TenantRequest{
				Tenant: "tenant3",
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sreConfigMapName,
					Namespace: sreNamespace,
				},
				Data: map[string]string{
					sreConfigName: initialConfig,
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      srePodName,
					Namespace: sreNamespace,
				},
			},
			expectedPodUpdate: false,
			expectedError:     false,
			expectedCode:      codes.OK,
			expectedConfig: `{
				"namespace":"orch_edgenode",
				"source":{
					"queryURI":"http://testurl:8181/prometheus",
					"mimirOrg":"tenant1|tenant2"
				},
				"collectors": null
			}`,
		},
		{
			name:   "unknown action",
			action: "unknown",
			tenantRequest: &pb.TenantRequest{
				Tenant: "tenant3",
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sreConfigMapName,
					Namespace: sreNamespace,
				},
				Data: map[string]string{
					sreConfigName: "{}",
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      srePodName,
					Namespace: sreNamespace,
				},
			},
			expectedPodUpdate: false,
			expectedError:     true,
			expectedCode:      codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewClientset()
			if test.configMap != nil {
				_, err := clientset.CoreV1().ConfigMaps(sreNamespace).Create(t.Context(), test.configMap, metav1.CreateOptions{})
				require.NoError(t, err, "Failed to create ConfigMap in fake clientset")
			}
			if test.pod != nil {
				_, err := clientset.CoreV1().Pods(sreNamespace).Create(t.Context(), test.pod, metav1.CreateOptions{})
				require.NoError(t, err, "Failed to create Pod in fake clientset")
			}

			server := &Server{
				gRPCPort:           "50051",
				configMapName:      sreConfigMapName,
				configName:         sreConfigName,
				podName:            srePodName,
				namespace:          sreNamespace,
				reloadEndpoint:     ts.URL + "/reload",
				configHashEndpoint: ts.URL + "/confighash",
				clientset:          clientset,
				grpcServer:         grpc.NewServer(),
			}

			_, err := server.processTenant(t.Context(), test.tenantRequest, test.action)
			if test.expectedError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				actualCode := st.Code()
				require.Equal(t, test.expectedCode, actualCode, "Expected code: %v, but got: %v", test.expectedCode, actualCode)
			} else {
				require.NoError(t, err)
				// Verify the config in the ConfigMap
				cm, err := clientset.CoreV1().ConfigMaps(sreNamespace).Get(t.Context(), sreConfigMapName, metav1.GetOptions{})
				require.NoError(t, err, "Failed to get ConfigMap from fake clientset")
				actualConfig := cm.Data[sreConfigName]
				require.JSONEq(t, test.expectedConfig, actualConfig, "Expected config: %v, but got: %v", test.expectedConfig, actualConfig)

				if test.expectedPodUpdate {
					// Verify that pod has annotation
					pod, err := clientset.CoreV1().Pods(sreNamespace).Get(t.Context(), srePodName, metav1.GetOptions{})
					require.NoError(t, err, "Failed to get Pod from fake clientset")
					require.NotEmpty(t, pod.Annotations["configMapHash"], "Empty configMapHash pod annotation")
				}
			}
		})
	}
}
