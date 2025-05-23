package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s-web-service/internal/k8s"
)

// ConnectK8sHandler handles the /connect-k8s endpoint
func (h *Handler) ConnectK8sHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate AWS configuration
	if err := h.config.ValidateAWSConfig(); err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("AWS configuration validation failed: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(h.config)
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to create Kubernetes client: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Test connection
	ctx := context.Background()
	if err := client.TestConnection(ctx); err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to connect to Kubernetes cluster: %v", err),
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get EKS details
	eksDetails := client.GetEKSDetails()

	response := map[string]interface{}{
		"status":            "success",
		"message":           "Successfully connected to Kubernetes cluster",
		"cluster_name":      eksDetails.ClusterName,
		"cluster_endpoint":  eksDetails.ClusterEndpoint,
		"region":            eksDetails.Region,
		"default_namespace": h.config.Kubernetes.DefaultNamespace,
	}

	json.NewEncoder(w).Encode(response)
}

// ListPodsHandler handles the /list-pods endpoint
func (h *Handler) ListPodsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	namespace := h.config.Kubernetes.DefaultNamespace
	if ns := r.URL.Query().Get("namespace"); ns != "" {
		namespace = ns
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(h.config)
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to create Kubernetes client: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// List pods
	ctx := context.Background()
	pods, err := client.GetClientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to list pods in namespace %s: %v", namespace, err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Format pod information
	var podList []map[string]interface{}
	for _, pod := range pods.Items {
		podInfo := map[string]interface{}{
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"status":    string(pod.Status.Phase),
			"node":      pod.Spec.NodeName,
			"created":   pod.CreationTimestamp.Time,
		}
		podList = append(podList, podInfo)
	}

	response := map[string]interface{}{
		"status":    "success",
		"namespace": namespace,
		"count":     len(podList),
		"pods":      podList,
	}

	json.NewEncoder(w).Encode(response)
}
