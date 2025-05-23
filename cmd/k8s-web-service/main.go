package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"k8s-web-service/internal/config"
	"k8s-web-service/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set default values if not configured
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Kubernetes.DefaultNamespace == "" {
		cfg.Kubernetes.DefaultNamespace = "default"
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Default namespace: %s", cfg.Kubernetes.DefaultNamespace)
	log.Printf("AWS region for EKS: %s", cfg.AWS.Region)

	// Create handlers
	h := handlers.New(cfg)

	// Setup routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":  "success",
			"message": "Kubernetes Web Service API",
			"version": "2.0.0",
			"server_info": map[string]interface{}{
				"host":     cfg.Server.Host,
				"port":     cfg.Server.Port,
				"base_url": fmt.Sprintf("http://%s:%s", cfg.Server.Host, cfg.Server.Port),
			},
			"endpoints": []map[string]interface{}{
				{
					"path":        "/connect-k8s",
					"method":      "GET",
					"description": "Test Kubernetes connection",
					"example_url": fmt.Sprintf("http://%s:%s/connect-k8s", cfg.Server.Host, cfg.Server.Port),
				},
				{
					"path":        "/list-pods",
					"method":      "GET",
					"description": "List pods in namespace",
					"parameters":  []string{"namespace (optional)"},
					"example_url": fmt.Sprintf("http://%s:%s/list-pods?namespace=%s", cfg.Server.Host, cfg.Server.Port, cfg.Kubernetes.DefaultNamespace),
				},
				{
					"path":        "/cluster-ca",
					"method":      "GET",
					"description": "Get cluster CA certificate",
					"example_url": fmt.Sprintf("http://%s:%s/cluster-ca", cfg.Server.Host, cfg.Server.Port),
				},
				{
					"path":        "/cluster-ca-expiry",
					"method":      "GET",
					"description": "Analyze cluster CA certificate expiry with detailed date information",
					"parameters":  []string{"warning_days (optional, default: 30)"},
					"example_url": fmt.Sprintf("http://%s:%s/cluster-ca-expiry?warning_days=365", cfg.Server.Host, cfg.Server.Port),
					"response_includes": []string{
						"formatted_dates", "time_remaining", "expiry_status", "validity_period",
					},
				},
				{
					"path":        "/pod-certificates",
					"method":      "GET",
					"description": "Analyze pod certificates (use ?detailed=true for expiry analysis)",
					"parameters":  []string{"namespace (optional)", "detailed (optional)", "warning_days (optional)"},
					"example_url": fmt.Sprintf("http://%s:%s/pod-certificates?detailed=true&warning_days=90", cfg.Server.Host, cfg.Server.Port),
				},
				{
					"path":        "/pod-certificates/{pod-name}",
					"method":      "GET",
					"description": "Detailed certificate analysis for specific pod",
					"parameters":  []string{"namespace (optional)", "warning_days (optional)"},
					"example_url": fmt.Sprintf("http://%s:%s/pod-certificates/example-pod?namespace=%s&warning_days=30", cfg.Server.Host, cfg.Server.Port, cfg.Kubernetes.DefaultNamespace),
				},
				{
					"path":        "/certificate-expiry",
					"method":      "GET",
					"description": "Certificate expiry analysis across namespace",
					"parameters":  []string{"namespace (optional)", "warning_days (optional)"},
					"example_url": fmt.Sprintf("http://%s:%s/certificate-expiry?namespace=%s&warning_days=60", cfg.Server.Host, cfg.Server.Port, cfg.Kubernetes.DefaultNamespace),
				},
				{
					"path":        "/debug",
					"method":      "GET",
					"description": "Debug AWS and Kubernetes configuration",
					"example_url": fmt.Sprintf("http://%s:%s/debug", cfg.Server.Host, cfg.Server.Port),
				},
				{
					"path":        "/test-k8s-auth",
					"method":      "GET",
					"description": "Test Kubernetes authentication",
					"example_url": fmt.Sprintf("http://%s:%s/test-k8s-auth", cfg.Server.Host, cfg.Server.Port),
				},
				{
					"path":        "/api-docs",
					"method":      "GET",
					"description": "Detailed API documentation with examples",
					"example_url": fmt.Sprintf("http://%s:%s/api-docs", cfg.Server.Host, cfg.Server.Port),
				},
			},
			"postman_tips": []string{
				"All endpoints return JSON responses",
				"Use query parameters to customize responses",
				"Set Content-Type: application/json in headers",
				"Check the /api-docs endpoint for detailed examples",
			},
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	})

	http.HandleFunc("/connect-k8s", h.ConnectK8sHandler)
	http.HandleFunc("/list-pods", h.ListPodsHandler)
	http.HandleFunc("/cluster-ca", h.ClusterCAHandler)
	http.HandleFunc("/cluster-ca-expiry", h.HandleClusterCACertificateExpiry)
	http.HandleFunc("/pod-certificates/", h.HandlePodCertificateDetails)
	http.HandleFunc("/pod-certificates", h.HandlePodCertificates)
	http.HandleFunc("/certificate-expiry", h.HandleCertificateExpiry)
	http.HandleFunc("/debug", h.DebugHandler)
	http.HandleFunc("/test-k8s-auth", h.TestK8sAuthHandler)
	http.HandleFunc("/api-docs", h.APIDocsHandler)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
