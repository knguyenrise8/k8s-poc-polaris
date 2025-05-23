package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s-web-service/internal/k8s"
)

// DebugHandler handles the /debug endpoint
func (h *Handler) DebugHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	debugInfo := map[string]interface{}{
		"status": "success",
	}

	// AWS Configuration Status
	awsConfigStatus := map[string]interface{}{
		"has_access_key":    h.config.AWS.AccessKeyID != "",
		"has_secret_key":    h.config.AWS.SecretAccessKey != "",
		"region":            h.config.AWS.Region,
		"validation_result": "unknown",
	}

	if err := h.config.ValidateAWSConfig(); err != nil {
		awsConfigStatus["validation_result"] = fmt.Sprintf("failed: %v", err)
	} else {
		awsConfigStatus["validation_result"] = "passed"
	}

	debugInfo["aws_config"] = awsConfigStatus

	// Try to get AWS caller identity
	client, err := k8s.NewClient(h.config)
	if err != nil {
		debugInfo["aws_identity"] = map[string]interface{}{
			"error": fmt.Sprintf("Failed to create client: %v", err),
		}
	} else {
		eksDetails := client.GetEKSDetails()
		debugInfo["kubeconfig_details"] = eksDetails
	}

	json.NewEncoder(w).Encode(debugInfo)
}

// TestK8sAuthHandler handles the /test-k8s-auth endpoint
func (h *Handler) TestK8sAuthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	results := map[string]interface{}{
		"status": "running_tests",
		"tests":  map[string]interface{}{},
	}

	// Test 1: AWS Configuration
	if err := h.config.ValidateAWSConfig(); err != nil {
		results["tests"].(map[string]interface{})["aws_config"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		results["status"] = "failed"
		json.NewEncoder(w).Encode(results)
		return
	}
	results["tests"].(map[string]interface{})["aws_config"] = map[string]interface{}{
		"status": "passed",
	}

	// Test 2: Create Kubernetes client
	client, err := k8s.NewClient(h.config)
	if err != nil {
		results["tests"].(map[string]interface{})["k8s_client_creation"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		results["status"] = "failed"
		json.NewEncoder(w).Encode(results)
		return
	}
	results["tests"].(map[string]interface{})["k8s_client_creation"] = map[string]interface{}{
		"status": "passed",
	}

	ctx := context.Background()

	// Test 3: List namespaces (basic cluster access)
	namespaces, err := client.GetClientset().CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		results["tests"].(map[string]interface{})["list_namespaces"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
	} else {
		results["tests"].(map[string]interface{})["list_namespaces"] = map[string]interface{}{
			"status": "passed",
			"count":  len(namespaces.Items),
		}
	}

	// Test 4: Get specific namespace
	targetNamespace := h.config.Kubernetes.DefaultNamespace
	_, err = client.GetClientset().CoreV1().Namespaces().Get(ctx, targetNamespace, metav1.GetOptions{})
	if err != nil {
		results["tests"].(map[string]interface{})["get_target_namespace"] = map[string]interface{}{
			"status":    "failed",
			"namespace": targetNamespace,
			"error":     err.Error(),
		}
	} else {
		results["tests"].(map[string]interface{})["get_target_namespace"] = map[string]interface{}{
			"status":    "passed",
			"namespace": targetNamespace,
		}
	}

	// Test 5: List pods in target namespace
	pods, err := client.GetClientset().CoreV1().Pods(targetNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		results["tests"].(map[string]interface{})["list_pods_target_namespace"] = map[string]interface{}{
			"status":    "failed",
			"namespace": targetNamespace,
			"error":     err.Error(),
		}
	} else {
		results["tests"].(map[string]interface{})["list_pods_target_namespace"] = map[string]interface{}{
			"status":    "passed",
			"namespace": targetNamespace,
			"count":     len(pods.Items),
		}
	}

	// Test 6: List pods in default namespace
	defaultPods, err := client.GetClientset().CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		results["tests"].(map[string]interface{})["list_pods_default_namespace"] = map[string]interface{}{
			"status":    "failed",
			"namespace": "default",
			"error":     err.Error(),
		}
	} else {
		results["tests"].(map[string]interface{})["list_pods_default_namespace"] = map[string]interface{}{
			"status":    "passed",
			"namespace": "default",
			"count":     len(defaultPods.Items),
		}
	}

	// Determine overall status
	allPassed := true
	for _, test := range results["tests"].(map[string]interface{}) {
		if testMap, ok := test.(map[string]interface{}); ok {
			if status, exists := testMap["status"]; exists && status != "passed" {
				allPassed = false
				break
			}
		}
	}

	if allPassed {
		results["status"] = "all_tests_passed"
	} else {
		results["status"] = "some_tests_failed"
	}

	json.NewEncoder(w).Encode(results)
}

// Helper functions

func isCertificateMount(mountPath string) bool {
	certPaths := []string{
		"/var/run/secrets/kubernetes.io/serviceaccount",
		"/etc/ssl",
		"/etc/certs",
		"/etc/pki",
		"/usr/share/ca-certificates",
		"/etc/ca-certificates",
	}

	for _, certPath := range certPaths {
		if strings.HasPrefix(mountPath, certPath) {
			return true
		}
	}

	// Check for common certificate file extensions in the path
	certExtensions := []string{".crt", ".pem", ".key", ".p12", ".jks", ".truststore"}
	for _, ext := range certExtensions {
		if strings.Contains(mountPath, ext) {
			return true
		}
	}

	return false
}

func getVolumeType(volume interface{}) string {
	// This is a simplified version - in reality you'd check all volume types
	v := volume.(interface{})
	switch {
	case v != nil:
		// Check various volume types
		return "unknown"
	default:
		return "unknown"
	}
}
