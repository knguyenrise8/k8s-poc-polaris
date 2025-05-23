package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"k8s-web-service/internal/k8s"
	"k8s-web-service/pkg/utils"
)

// ClusterCAHandler handles the /cluster-ca endpoint
func (h *Handler) ClusterCAHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get kubeconfig path
	kubeconfigPath := k8s.GetKubeconfigPath()
	if kubeconfigPath == "" {
		response := map[string]interface{}{
			"status": "error",
			"error":  "Could not determine kubeconfig path",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get cluster CA
	clusterCA, err := k8s.GetClusterCA(kubeconfigPath)
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to get cluster CA: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create Kubernetes client to get additional details
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

	eksDetails := client.GetEKSDetails()

	response := map[string]interface{}{
		"status":      "success",
		"message":     "Retrieved cluster CA certificate",
		"description": "This is the CA certificate that all pods use to verify the Kubernetes API server",
		"ca_certificate": map[string]interface{}{
			"pem_content": clusterCA,
			"length":      len(clusterCA),
		},
		"source": "kubeconfig certificate-authority-data",
		"usage":  "Mounted at /var/run/secrets/kubernetes.io/serviceaccount/ca.crt in every pod",
		"cluster_info": map[string]interface{}{
			"region":           eksDetails.Region,
			"cluster_endpoint": eksDetails.ClusterEndpoint,
			"cluster_name":     eksDetails.ClusterName,
		},
		"notes": []string{
			"This certificate is automatically mounted in every pod",
			"Pods use this to verify the identity of the Kubernetes API server",
			"This is different from client certificates used for authentication",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// HandleClusterCACertificateExpiry handles requests for cluster CA certificate expiry analysis
func (h *Handler) HandleClusterCACertificateExpiry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get warning days from query parameter (default 30 days)
	warningDaysStr := r.URL.Query().Get("warning_days")
	warningDays := 30
	if warningDaysStr != "" {
		if days, err := strconv.Atoi(warningDaysStr); err == nil && days > 0 {
			warningDays = days
		}
	}

	// Get kubeconfig path
	kubeconfigPath := k8s.GetKubeconfigPath()
	if kubeconfigPath == "" {
		response := map[string]interface{}{
			"status": "error",
			"error":  "Could not determine kubeconfig path",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get cluster CA
	clusterCA, err := k8s.GetClusterCA(kubeconfigPath)
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to get cluster CA: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse the cluster CA certificate and get expiry information
	certSource, err := k8s.GetClusterCACertificateInfo(clusterCA)
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to parse cluster CA certificate: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get expiry warnings
	certSources := map[string]*k8s.CertificateSource{
		"cluster-ca": certSource,
	}
	warnings := k8s.GetCertificateExpiryWarnings(certSources, warningDays)

	// Create enhanced certificate info with formatted dates
	var enhancedCertInfo map[string]interface{}
	if len(certSource.Certificates) > 0 {
		cert := certSource.Certificates[0]

		// Calculate time remaining in different units
		timeUntilExpiry := cert.NotAfter.Sub(time.Now())
		years := int(timeUntilExpiry.Hours() / (24 * 365))
		months := int(timeUntilExpiry.Hours() / (24 * 30))
		weeks := int(timeUntilExpiry.Hours() / (24 * 7))

		enhancedCertInfo = map[string]interface{}{
			"subject":       cert.Subject,
			"issuer":        cert.Issuer,
			"serial_number": cert.SerialNumber,
			"is_ca":         cert.IsCA,
			"is_expired":    cert.IsExpired,
			"validity_period": map[string]interface{}{
				"not_before":           cert.NotBefore,
				"not_after":            cert.NotAfter,
				"not_before_formatted": cert.NotBefore.Format("January 2, 2006 at 3:04 PM MST"),
				"not_after_formatted":  cert.NotAfter.Format("January 2, 2006 at 3:04 PM MST"),
				"valid_for_days":       int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24),
			},
			"expiry_info": map[string]interface{}{
				"days_until_expiry":   cert.DaysUntilExp,
				"weeks_until_expiry":  weeks,
				"months_until_expiry": months,
				"years_until_expiry":  years,
				"expires_on":          cert.NotAfter.Format("January 2, 2006"),
				"expires_on_weekday":  cert.NotAfter.Format("Monday, January 2, 2006"),
				"time_remaining":      formatDuration(timeUntilExpiry),
			},
			"dns_names":    cert.DNSNames,
			"ip_addresses": cert.IPAddresses,
			"key_usage":    cert.KeyUsage,
		}
	}

	// Create detailed response
	response := map[string]interface{}{
		"status":        "success",
		"message":       "Cluster CA certificate expiry analysis",
		"warning_days":  warningDays,
		"analysis_date": time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		"certificate_info": map[string]interface{}{
			"source":        certSource,
			"warnings":      warnings,
			"total_certs":   len(certSource.Certificates),
			"enhanced_info": enhancedCertInfo,
		},
		"summary": map[string]interface{}{
			"certificates_analyzed": len(certSource.Certificates),
			"warnings_found":        len(warnings),
			"expires_within_days":   warningDays,
			"status_summary":        getExpiryStatusSummary(certSource.Certificates, warningDays),
		},
		"notes": []string{
			"This is the Kubernetes cluster CA certificate used to verify the API server",
			"All pods automatically receive this certificate at /var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			fmt.Sprintf("Analysis performed with %d day warning threshold", warningDays),
			"Use ?warning_days=N to customize the warning threshold",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "Expired"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24

	if days > 365 {
		years := days / 365
		remainingDays := days % 365
		if remainingDays > 0 {
			return fmt.Sprintf("%d years, %d days", years, remainingDays)
		}
		return fmt.Sprintf("%d years", years)
	} else if days > 30 {
		months := days / 30
		remainingDays := days % 30
		if remainingDays > 0 {
			return fmt.Sprintf("%d months, %d days", months, remainingDays)
		}
		return fmt.Sprintf("%d months", months)
	} else if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%d days, %d hours", days, hours)
		}
		return fmt.Sprintf("%d days", days)
	} else {
		return fmt.Sprintf("%d hours", hours)
	}
}

// getExpiryStatusSummary provides a summary of certificate expiry status
func getExpiryStatusSummary(certs []*utils.CertificateInfo, warningDays int) string {
	if len(certs) == 0 {
		return "No certificates found"
	}

	for _, cert := range certs {
		if cert.IsExpired {
			return "EXPIRED"
		}
		if cert.DaysUntilExp <= warningDays {
			return fmt.Sprintf("EXPIRES SOON (%d days)", cert.DaysUntilExp)
		}
	}

	// Find the certificate that expires soonest
	minDays := certs[0].DaysUntilExp
	for _, cert := range certs {
		if cert.DaysUntilExp < minDays {
			minDays = cert.DaysUntilExp
		}
	}

	return fmt.Sprintf("VALID (%d days remaining)", minDays)
}
