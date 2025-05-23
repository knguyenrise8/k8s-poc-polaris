package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIDocsHandler provides comprehensive API documentation with examples
func (h *Handler) APIDocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	baseURL := fmt.Sprintf("http://%s:%s", h.config.Server.Host, h.config.Server.Port)

	response := map[string]interface{}{
		"status":       "success",
		"message":      "Kubernetes Web Service API Documentation",
		"version":      "2.0.0",
		"base_url":     baseURL,
		"generated_at": time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		"endpoints": map[string]interface{}{
			"connect_k8s": map[string]interface{}{
				"url":         fmt.Sprintf("%s/connect-k8s", baseURL),
				"method":      "GET",
				"description": "Test Kubernetes cluster connectivity and return cluster information",
				"parameters":  "None",
				"example_response": map[string]interface{}{
					"status":            "success",
					"message":           "Successfully connected to Kubernetes cluster",
					"cluster_name":      "your-cluster",
					"cluster_endpoint":  "https://...",
					"region":            "us-gov-west-1",
					"default_namespace": h.config.Kubernetes.DefaultNamespace,
				},
			},
			"list_pods": map[string]interface{}{
				"url":         fmt.Sprintf("%s/list-pods", baseURL),
				"method":      "GET",
				"description": "List all pods in a namespace with their status and details",
				"parameters": map[string]string{
					"namespace": "Target namespace (optional, defaults to configured namespace)",
				},
				"example_urls": []string{
					fmt.Sprintf("%s/list-pods", baseURL),
					fmt.Sprintf("%s/list-pods?namespace=%s", baseURL, h.config.Kubernetes.DefaultNamespace),
					fmt.Sprintf("%s/list-pods?namespace=default", baseURL),
				},
				"example_response": map[string]interface{}{
					"status":    "success",
					"namespace": h.config.Kubernetes.DefaultNamespace,
					"count":     5,
					"pods": []map[string]interface{}{
						{
							"name":      "example-pod-123",
							"namespace": h.config.Kubernetes.DefaultNamespace,
							"status":    "Running",
							"node":      "node-1",
							"created":   "2024-01-01T00:00:00Z",
						},
					},
				},
			},
			"cluster_ca": map[string]interface{}{
				"url":         fmt.Sprintf("%s/cluster-ca", baseURL),
				"method":      "GET",
				"description": "Retrieve the Kubernetes cluster CA certificate",
				"parameters":  "None",
				"example_response": map[string]interface{}{
					"status":  "success",
					"message": "Retrieved cluster CA certificate",
					"ca_certificate": map[string]interface{}{
						"pem_content": "-----BEGIN CERTIFICATE-----...",
						"length":      1099,
					},
					"cluster_info": map[string]interface{}{
						"region":           "us-gov-west-1",
						"cluster_endpoint": "https://...",
						"cluster_name":     "your-cluster",
					},
				},
			},
			"cluster_ca_expiry": map[string]interface{}{
				"url":         fmt.Sprintf("%s/cluster-ca-expiry", baseURL),
				"method":      "GET",
				"description": "Analyze cluster CA certificate expiry with detailed date information",
				"parameters": map[string]string{
					"warning_days": "Number of days before expiry to warn (optional, default: 30)",
				},
				"example_urls": []string{
					fmt.Sprintf("%s/cluster-ca-expiry", baseURL),
					fmt.Sprintf("%s/cluster-ca-expiry?warning_days=365", baseURL),
					fmt.Sprintf("%s/cluster-ca-expiry?warning_days=90", baseURL),
				},
				"response_features": []string{
					"Formatted expiry dates (human-readable)",
					"Time remaining in years/months/days",
					"Certificate validity period",
					"Expiry status summary",
					"Analysis timestamp",
				},
				"example_response": map[string]interface{}{
					"status":        "success",
					"analysis_date": "May 23, 2025 at 2:19 PM CDT",
					"certificate_info": map[string]interface{}{
						"enhanced_info": map[string]interface{}{
							"validity_period": map[string]interface{}{
								"not_before_formatted": "May 4, 2023 at 5:37 PM UTC",
								"not_after_formatted":  "May 1, 2033 at 5:37 PM UTC",
								"valid_for_days":       3650,
							},
							"expiry_info": map[string]interface{}{
								"days_until_expiry":  2899,
								"expires_on":         "May 1, 2033",
								"expires_on_weekday": "Sunday, May 1, 2033",
								"time_remaining":     "7 years, 344 days",
							},
						},
					},
					"summary": map[string]interface{}{
						"status_summary": "VALID (2899 days remaining)",
					},
				},
			},
			"pod_certificates": map[string]interface{}{
				"url":         fmt.Sprintf("%s/pod-certificates", baseURL),
				"method":      "GET",
				"description": "Analyze certificate mounts and sources in pods",
				"parameters": map[string]string{
					"namespace":    "Target namespace (optional)",
					"detailed":     "Include certificate expiry analysis (true/false, optional)",
					"warning_days": "Warning threshold in days (optional, default: 30)",
				},
				"example_urls": []string{
					fmt.Sprintf("%s/pod-certificates", baseURL),
					fmt.Sprintf("%s/pod-certificates?detailed=true", baseURL),
					fmt.Sprintf("%s/pod-certificates?detailed=true&warning_days=90", baseURL),
					fmt.Sprintf("%s/pod-certificates?namespace=default&detailed=true", baseURL),
				},
			},
			"pod_certificate_details": map[string]interface{}{
				"url":         fmt.Sprintf("%s/pod-certificates/{pod-name}", baseURL),
				"method":      "GET",
				"description": "Detailed certificate analysis for a specific pod",
				"parameters": map[string]string{
					"pod-name":     "Name of the pod (required in URL path)",
					"namespace":    "Target namespace (optional)",
					"warning_days": "Warning threshold in days (optional, default: 30)",
				},
				"example_urls": []string{
					fmt.Sprintf("%s/pod-certificates/example-pod", baseURL),
					fmt.Sprintf("%s/pod-certificates/example-pod?namespace=%s", baseURL, h.config.Kubernetes.DefaultNamespace),
					fmt.Sprintf("%s/pod-certificates/example-pod?warning_days=60", baseURL),
				},
			},
			"certificate_expiry": map[string]interface{}{
				"url":         fmt.Sprintf("%s/certificate-expiry", baseURL),
				"method":      "GET",
				"description": "Certificate expiry analysis across all pods in a namespace",
				"parameters": map[string]string{
					"namespace":    "Target namespace (optional)",
					"warning_days": "Warning threshold in days (optional, default: 30)",
				},
				"example_urls": []string{
					fmt.Sprintf("%s/certificate-expiry", baseURL),
					fmt.Sprintf("%s/certificate-expiry?namespace=%s&warning_days=60", baseURL, h.config.Kubernetes.DefaultNamespace),
				},
			},
			"debug": map[string]interface{}{
				"url":         fmt.Sprintf("%s/debug", baseURL),
				"method":      "GET",
				"description": "Debug AWS and Kubernetes configuration",
				"parameters":  "None",
				"use_case":    "Troubleshooting connectivity issues",
			},
			"test_k8s_auth": map[string]interface{}{
				"url":         fmt.Sprintf("%s/test-k8s-auth", baseURL),
				"method":      "GET",
				"description": "Comprehensive Kubernetes authentication testing",
				"parameters":  "None",
				"use_case":    "Verify permissions and access levels",
			},
		},
		"postman_collection": map[string]interface{}{
			"info": map[string]interface{}{
				"name":        "Kubernetes Web Service API",
				"description": "Collection for testing Kubernetes certificate analysis endpoints",
				"version":     "2.0.0",
			},
			"quick_start": []string{
				"1. Import this response as a Postman collection",
				"2. Set base_url as an environment variable",
				"3. Start with /connect-k8s to test connectivity",
				"4. Use /cluster-ca-expiry for certificate date analysis",
				"5. Try /pod-certificates?detailed=true for comprehensive analysis",
			},
			"common_headers": map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
		},
		"configuration": map[string]interface{}{
			"default_namespace": h.config.Kubernetes.DefaultNamespace,
			"aws_region":        h.config.AWS.Region,
			"cluster_name":      h.config.Kubernetes.ClusterName,
		},
		"notes": []string{
			"All endpoints return JSON responses",
			"Query parameters are optional unless specified",
			"Date information includes multiple formats for convenience",
			"Use warning_days parameter to customize expiry thresholds",
			"The detailed=true parameter provides comprehensive certificate analysis",
		},
	}

	json.NewEncoder(w).Encode(response)
}
