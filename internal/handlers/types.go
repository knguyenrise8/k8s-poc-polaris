package handlers

import "k8s-web-service/internal/k8s"

// VolumeMount represents a volume mount in a pod
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	ReadOnly  bool   `json:"read_only"`
	Container string `json:"container"`
}

// Volume represents a volume in a pod
type Volume struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source,omitempty"`
}

// ClusterCAInfo represents cluster CA certificate information
type ClusterCAInfo struct {
	Description string `json:"description"`
	Length      int    `json:"length"`
	Source      string `json:"source"`
}

// PodCertificatesResponse represents the response for pod certificates with expiry info
type PodCertificatesResponse struct {
	Status          string        `json:"status"`
	Message         string        `json:"message"`
	TargetNamespace string        `json:"target_namespace"`
	ClusterCAInfo   ClusterCAInfo `json:"cluster_ca_info"`
	Pods            []PodCertInfo `json:"pods"`
	ExpiryWarnings  []string      `json:"expiry_warnings,omitempty"`
	Notes           []string      `json:"notes"`
}

// PodCertInfo represents certificate information for a pod with expiry details
type PodCertInfo struct {
	Name               string                            `json:"name"`
	Namespace          string                            `json:"namespace"`
	VolumeMounts       []VolumeMount                     `json:"volume_mounts"`
	Volumes            []Volume                          `json:"volumes"`
	CertificateSources map[string]*k8s.CertificateSource `json:"certificate_sources,omitempty"`
	ExpiryWarnings     []string                          `json:"expiry_warnings,omitempty"`
}
