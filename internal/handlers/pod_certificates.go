package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s-web-service/internal/k8s"
)

// PodCertificatesHandler handles the /pod-certificates endpoint
func (h *Handler) PodCertificatesHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get cluster CA
	eksDetails := client.GetEKSDetails()

	// Analyze pods for certificate information
	var podCertificates []map[string]interface{}
	for _, pod := range pods.Items {
		podInfo := map[string]interface{}{
			"name":      pod.Name,
			"namespace": pod.Namespace,
		}

		// Analyze volume mounts for certificates
		var volumeMounts []map[string]interface{}
		var volumes []map[string]interface{}

		for _, container := range pod.Spec.Containers {
			for _, mount := range container.VolumeMounts {
				if isCertificateMount(mount.MountPath) {
					volumeMounts = append(volumeMounts, map[string]interface{}{
						"container":   container.Name,
						"mount_path":  mount.MountPath,
						"volume_name": mount.Name,
						"read_only":   mount.ReadOnly,
					})
				}
			}
		}

		// Analyze volumes for certificate sources
		for _, volume := range pod.Spec.Volumes {
			volumeInfo := map[string]interface{}{
				"name": volume.Name,
				"type": getVolumeType(volume),
			}

			if volume.Secret != nil {
				volumeInfo["secret_name"] = volume.Secret.SecretName
			}
			if volume.ConfigMap != nil {
				volumeInfo["configmap_name"] = volume.ConfigMap.Name
			}

			volumes = append(volumes, volumeInfo)
		}

		podInfo["volume_mounts"] = volumeMounts
		podInfo["volumes"] = volumes
		podCertificates = append(podCertificates, podInfo)
	}

	response := map[string]interface{}{
		"status":           "success",
		"message":          fmt.Sprintf("Retrieved certificate information for %d pods in namespace '%s'", len(podCertificates), namespace),
		"target_namespace": namespace,
		"cluster_ca_info": map[string]interface{}{
			"description": "The cluster CA certificate used by your kubeconfig",
			"length":      len(eksDetails.ClusterCA),
			"source":      "kubeconfig certificate-authority-data",
		},
		"pods": podCertificates,
		"notes": []string{
			"All pods automatically receive the Kubernetes cluster CA at /var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			"Additional certificates may be mounted via secrets, configmaps, or projected volumes",
			"To extract actual certificate content, you need to exec into the pod or read the secret/configmap directly",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// HandlePodCertificates handles requests for pod certificate information with expiry analysis
func (h *Handler) HandlePodCertificates(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get namespace from query parameter or use default
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = h.config.Kubernetes.DefaultNamespace
	}

	// Get warning days from query parameter (default 30 days)
	warningDaysStr := r.URL.Query().Get("warning_days")
	warningDays := 30
	if warningDaysStr != "" {
		if days, err := strconv.Atoi(warningDaysStr); err == nil && days > 0 {
			warningDays = days
		}
	}

	// Get detailed analysis flag
	detailed := r.URL.Query().Get("detailed") == "true"

	// Create Kubernetes client
	client, err := k8s.NewClient(h.config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Kubernetes client: %v", err), http.StatusInternalServerError)
		return
	}

	pods, err := client.GetClientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	eksDetails := client.GetEKSDetails()

	var podCertInfos []PodCertInfo
	var allExpiryWarnings []string

	for _, pod := range pods.Items {
		podInfo := PodCertInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}

		// Get volume mounts and volumes (existing logic)
		for _, container := range pod.Spec.Containers {
			for _, mount := range container.VolumeMounts {
				podInfo.VolumeMounts = append(podInfo.VolumeMounts, VolumeMount{
					Name:      mount.Name,
					MountPath: mount.MountPath,
					ReadOnly:  mount.ReadOnly,
					Container: container.Name,
				})
			}
		}

		for _, volume := range pod.Spec.Volumes {
			volumeInfo := Volume{
				Name: volume.Name,
				Type: getVolumeType(volume),
			}

			if volume.Secret != nil {
				volumeInfo.Source = volume.Secret.SecretName
			} else if volume.ConfigMap != nil {
				volumeInfo.Source = volume.ConfigMap.Name
			} else if volume.Projected != nil {
				volumeInfo.Source = "projected"
			} else if volume.EmptyDir != nil {
				volumeInfo.Source = "emptyDir"
			}

			podInfo.Volumes = append(podInfo.Volumes, volumeInfo)
		}

		// If detailed analysis is requested, extract and analyze certificates
		if detailed {
			certSources, err := k8s.AnalyzePodCertificates(ctx, client, namespace, pod.Name)
			if err == nil {
				podInfo.CertificateSources = certSources

				// Get expiry warnings for this pod
				warnings := k8s.GetCertificateExpiryWarnings(certSources, warningDays)
				if len(warnings) > 0 {
					podInfo.ExpiryWarnings = warnings
					for _, warning := range warnings {
						allExpiryWarnings = append(allExpiryWarnings, fmt.Sprintf("Pod %s: %s", pod.Name, warning))
					}
				}
			}
		}

		podCertInfos = append(podCertInfos, podInfo)
	}

	response := PodCertificatesResponse{
		Status:          "success",
		Message:         fmt.Sprintf("Retrieved certificate information for %d pods in namespace '%s'", len(pods.Items), namespace),
		TargetNamespace: namespace,
		ClusterCAInfo: ClusterCAInfo{
			Description: "The cluster CA certificate used by your kubeconfig",
			Length:      len(eksDetails.ClusterCA),
			Source:      "kubeconfig certificate-authority-data",
		},
		Pods:           podCertInfos,
		ExpiryWarnings: allExpiryWarnings,
		Notes: []string{
			"All pods automatically receive the Kubernetes cluster CA at /var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			"Additional certificates may be mounted via secrets, configmaps, or projected volumes",
			"To extract actual certificate content, you need to exec into the pod or read the secret/configmap directly",
		},
	}

	if detailed {
		response.Notes = append(response.Notes,
			fmt.Sprintf("Certificate expiry analysis performed with %d day warning threshold", warningDays),
			"Use ?detailed=true&warning_days=N to customize the warning threshold",
		)
	} else {
		response.Notes = append(response.Notes, "Use ?detailed=true to include certificate expiry analysis")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandlePodCertificateDetails handles requests for detailed certificate analysis of a specific pod
func (h *Handler) HandlePodCertificateDetails(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get pod name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 || pathParts[2] == "" {
		http.Error(w, "Pod name is required in URL path: /pod-certificates/{pod-name}", http.StatusBadRequest)
		return
	}
	podName := pathParts[2]

	// Get namespace from query parameter or use default
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = h.config.Kubernetes.DefaultNamespace
	}

	// Get warning days from query parameter (default 30 days)
	warningDaysStr := r.URL.Query().Get("warning_days")
	warningDays := 30
	if warningDaysStr != "" {
		if days, err := strconv.Atoi(warningDaysStr); err == nil && days > 0 {
			warningDays = days
		}
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(h.config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Kubernetes client: %v", err), http.StatusInternalServerError)
		return
	}

	// Analyze certificates for the specific pod
	certSources, err := k8s.AnalyzePodCertificates(ctx, client, namespace, podName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze certificates for pod %s: %v", podName, err), http.StatusInternalServerError)
		return
	}

	// Get expiry warnings
	warnings := k8s.GetCertificateExpiryWarnings(certSources, warningDays)

	response := map[string]interface{}{
		"status":              "success",
		"message":             fmt.Sprintf("Certificate analysis for pod '%s' in namespace '%s'", podName, namespace),
		"pod_name":            podName,
		"namespace":           namespace,
		"warning_days":        warningDays,
		"certificate_sources": certSources,
		"expiry_warnings":     warnings,
		"summary": map[string]interface{}{
			"total_sources":      len(certSources),
			"total_certificates": getTotalCertificateCount(certSources),
			"warnings_count":     len(warnings),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCertificateExpiry handles requests for certificate expiry analysis across the namespace
func (h *Handler) HandleCertificateExpiry(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get namespace from query parameter or use default
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = h.config.Kubernetes.DefaultNamespace
	}

	// Get warning days from query parameter (default 30 days)
	warningDaysStr := r.URL.Query().Get("warning_days")
	warningDays := 30
	if warningDaysStr != "" {
		if days, err := strconv.Atoi(warningDaysStr); err == nil && days > 0 {
			warningDays = days
		}
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(h.config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Kubernetes client: %v", err), http.StatusInternalServerError)
		return
	}

	// Get pods in the namespace
	pods, err := client.GetClientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	type PodExpiryInfo struct {
		PodName      string                            `json:"pod_name"`
		CertSources  map[string]*k8s.CertificateSource `json:"certificate_sources"`
		Warnings     []string                          `json:"warnings"`
		WarningCount int                               `json:"warning_count"`
		CertCount    int                               `json:"certificate_count"`
	}

	var podExpiryInfos []PodExpiryInfo
	var allWarnings []string
	totalCerts := 0
	totalWarnings := 0

	for _, pod := range pods.Items {
		certSources, err := k8s.AnalyzePodCertificates(ctx, client, namespace, pod.Name)
		if err != nil {
			continue // Skip pods with errors
		}

		warnings := k8s.GetCertificateExpiryWarnings(certSources, warningDays)
		certCount := getTotalCertificateCount(certSources)

		if len(warnings) > 0 || certCount > 0 {
			podInfo := PodExpiryInfo{
				PodName:      pod.Name,
				CertSources:  certSources,
				Warnings:     warnings,
				WarningCount: len(warnings),
				CertCount:    certCount,
			}
			podExpiryInfos = append(podExpiryInfos, podInfo)

			for _, warning := range warnings {
				allWarnings = append(allWarnings, fmt.Sprintf("Pod %s: %s", pod.Name, warning))
			}
		}

		totalCerts += certCount
		totalWarnings += len(warnings)
	}

	response := map[string]interface{}{
		"status":       "success",
		"message":      fmt.Sprintf("Certificate expiry analysis for namespace '%s'", namespace),
		"namespace":    namespace,
		"warning_days": warningDays,
		"summary": map[string]interface{}{
			"total_pods_analyzed":    len(pods.Items),
			"pods_with_certificates": len(podExpiryInfos),
			"total_certificates":     totalCerts,
			"total_warnings":         totalWarnings,
		},
		"pod_expiry_info": podExpiryInfos,
		"all_warnings":    allWarnings,
		"notes": []string{
			fmt.Sprintf("Analysis performed with %d day warning threshold", warningDays),
			"Use ?warning_days=N to customize the warning threshold",
			"Only pods with certificates or warnings are included in the results",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getTotalCertificateCount counts total certificates across all sources
func getTotalCertificateCount(certSources map[string]*k8s.CertificateSource) int {
	total := 0
	for _, source := range certSources {
		total += len(source.Certificates)
	}
	return total
}
