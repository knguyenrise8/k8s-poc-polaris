package k8s

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s-web-service/pkg/utils"
)

// CertificateSource represents where a certificate comes from
type CertificateSource struct {
	Type         string                   `json:"type"`          // "secret", "configmap", "cluster-ca"
	Name         string                   `json:"name"`          // resource name
	Namespace    string                   `json:"namespace"`     // resource namespace
	Key          string                   `json:"key,omitempty"` // key within the resource
	Certificates []*utils.CertificateInfo `json:"certificates"`
	Error        string                   `json:"error,omitempty"`
}

// ExtractCertificatesFromSecret extracts certificates from a Kubernetes secret
func ExtractCertificatesFromSecret(ctx context.Context, clientset *kubernetes.Clientset, namespace, secretName string) (*CertificateSource, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return &CertificateSource{
			Type:      "secret",
			Name:      secretName,
			Namespace: namespace,
			Error:     fmt.Sprintf("Failed to get secret: %v", err),
		}, err
	}

	source := &CertificateSource{
		Type:      "secret",
		Name:      secretName,
		Namespace: namespace,
	}

	// Common certificate keys to check
	certKeys := []string{
		"tls.crt", "tls.cert", "cert.pem", "certificate.pem", "ca.crt", "ca.pem",
		"client.crt", "server.crt", "cert", "certificate", "ca-bundle.crt",
		"ca-bundle.pem", "root-ca.pem", "intermediate-ca.pem",
	}

	var allCerts []*utils.CertificateInfo

	for _, key := range certKeys {
		if certData, exists := secret.Data[key]; exists {
			certString := string(certData)

			// Try to parse as a single certificate first
			if cert, err := utils.ParseCertificate(certString); err == nil {
				cert.Subject = fmt.Sprintf("%s (from %s)", cert.Subject, key)
				allCerts = append(allCerts, cert)
				continue
			}

			// Try to parse as a certificate bundle
			if certs, err := utils.ParseCertificateBundle(certString); err == nil {
				for _, cert := range certs {
					cert.Subject = fmt.Sprintf("%s (from %s)", cert.Subject, key)
					allCerts = append(allCerts, cert)
				}
			}
		}
	}

	source.Certificates = allCerts
	return source, nil
}

// ExtractCertificatesFromConfigMap extracts certificates from a Kubernetes configmap
func ExtractCertificatesFromConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace, configMapName string) (*CertificateSource, error) {
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return &CertificateSource{
			Type:      "configmap",
			Name:      configMapName,
			Namespace: namespace,
			Error:     fmt.Sprintf("Failed to get configmap: %v", err),
		}, err
	}

	source := &CertificateSource{
		Type:      "configmap",
		Name:      configMapName,
		Namespace: namespace,
	}

	// Common certificate keys to check
	certKeys := []string{
		"ca.crt", "ca.pem", "ca-bundle.crt", "ca-bundle.pem", "root-ca.pem",
		"intermediate-ca.pem", "tls.crt", "tls.cert", "cert.pem", "certificate.pem",
		"client.crt", "server.crt", "cert", "certificate",
	}

	var allCerts []*utils.CertificateInfo

	// Check both Data and BinaryData
	for _, key := range certKeys {
		var certString string

		// Check in Data first
		if certData, exists := configMap.Data[key]; exists {
			certString = certData
		} else if certData, exists := configMap.BinaryData[key]; exists {
			certString = string(certData)
		} else {
			continue
		}

		// Try to parse as a single certificate first
		if cert, err := utils.ParseCertificate(certString); err == nil {
			cert.Subject = fmt.Sprintf("%s (from %s)", cert.Subject, key)
			allCerts = append(allCerts, cert)
			continue
		}

		// Try to parse as a certificate bundle
		if certs, err := utils.ParseCertificateBundle(certString); err == nil {
			for _, cert := range certs {
				cert.Subject = fmt.Sprintf("%s (from %s)", cert.Subject, key)
				allCerts = append(allCerts, cert)
			}
		}
	}

	source.Certificates = allCerts
	return source, nil
}

// GetClusterCACertificateInfo parses the cluster CA certificate and returns its info
func GetClusterCACertificateInfo(clusterCA string) (*CertificateSource, error) {
	source := &CertificateSource{
		Type: "cluster-ca",
		Name: "kubernetes-cluster-ca",
	}

	if clusterCA == "" {
		source.Error = "No cluster CA certificate available"
		return source, fmt.Errorf("no cluster CA certificate")
	}

	// Try to parse as a single certificate first
	if cert, err := utils.ParseCertificate(clusterCA); err == nil {
		cert.Subject = fmt.Sprintf("%s (Kubernetes Cluster CA)", cert.Subject)
		source.Certificates = []*utils.CertificateInfo{cert}
		return source, nil
	}

	// Try to parse as a certificate bundle
	if certs, err := utils.ParseCertificateBundle(clusterCA); err == nil {
		for _, cert := range certs {
			cert.Subject = fmt.Sprintf("%s (Kubernetes Cluster CA)", cert.Subject)
		}
		source.Certificates = certs
		return source, nil
	}

	source.Error = "Failed to parse cluster CA certificate"
	return source, fmt.Errorf("failed to parse cluster CA certificate")
}

// AnalyzePodCertificates analyzes all certificates in a pod and returns detailed information
func AnalyzePodCertificates(ctx context.Context, client *Client, namespace, podName string) (map[string]*CertificateSource, error) {
	clientset := client.GetClientset()

	// Get the pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	certSources := make(map[string]*CertificateSource)

	// Add cluster CA certificate
	eksDetails := client.GetEKSDetails()
	if clusterCAInfo, err := GetClusterCACertificateInfo(eksDetails.ClusterCA); err == nil {
		certSources["cluster-ca"] = clusterCAInfo
	}

	// Analyze volumes for certificate sources
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			secretName := volume.Secret.SecretName
			key := fmt.Sprintf("secret-%s", secretName)

			if source, err := ExtractCertificatesFromSecret(ctx, clientset, namespace, secretName); err == nil {
				certSources[key] = source
			} else {
				certSources[key] = &CertificateSource{
					Type:      "secret",
					Name:      secretName,
					Namespace: namespace,
					Error:     err.Error(),
				}
			}
		}

		if volume.ConfigMap != nil {
			configMapName := volume.ConfigMap.Name
			key := fmt.Sprintf("configmap-%s", configMapName)

			if source, err := ExtractCertificatesFromConfigMap(ctx, clientset, namespace, configMapName); err == nil {
				certSources[key] = source
			} else {
				certSources[key] = &CertificateSource{
					Type:      "configmap",
					Name:      configMapName,
					Namespace: namespace,
					Error:     err.Error(),
				}
			}
		}
	}

	return certSources, nil
}

// GetCertificateExpiryWarnings returns warnings for certificates expiring soon
func GetCertificateExpiryWarnings(certSources map[string]*CertificateSource, warningDays int) []string {
	var allWarnings []string

	for sourceName, source := range certSources {
		if len(source.Certificates) > 0 {
			warnings := utils.ValidateCertificateExpiry(source.Certificates, warningDays)
			for _, warning := range warnings {
				allWarnings = append(allWarnings, fmt.Sprintf("[%s] %s", sourceName, warning))
			}
		}
	}

	return allWarnings
}

// isCertificateKey checks if a key name suggests it contains certificate data
func isCertificateKey(key string) bool {
	key = strings.ToLower(key)
	certIndicators := []string{
		"crt", "cert", "certificate", "pem", "ca", "tls", "ssl",
		"root", "intermediate", "bundle", "chain",
	}

	for _, indicator := range certIndicators {
		if strings.Contains(key, indicator) {
			return true
		}
	}

	return false
}
