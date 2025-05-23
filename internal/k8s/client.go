package k8s

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s-web-service/internal/auth"
	"k8s-web-service/internal/config"
)

// KubeConfigEKSDetails contains EKS-specific details from kubeconfig
type KubeConfigEKSDetails struct {
	ClusterName     string `json:"cluster_name"`
	ClusterEndpoint string `json:"cluster_endpoint"`
	ClusterCA       string `json:"cluster_ca"`
	Region          string `json:"region"`
	RoleARN         string `json:"role_arn,omitempty"`
}

// Client wraps the Kubernetes client with additional functionality
type Client struct {
	clientset  *kubernetes.Clientset
	config     *rest.Config
	appConfig  *config.Config
	tokenGen   *auth.EKSTokenGenerator
	eksDetails *KubeConfigEKSDetails
}

// NewClient creates a new Kubernetes client
func NewClient(cfg *config.Config) (*Client, error) {
	// Get kubeconfig path
	kubeconfigPath := getKubeconfigPath()

	// Parse kubeconfig for EKS details
	eksDetails, err := parseKubeConfigForEKS(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig for EKS details: %w", err)
	}

	// Create token generator
	tokenGen := auth.NewEKSTokenGenerator(cfg)

	// Generate EKS token - try aws-iam-authenticator first for better compatibility
	token, err := tokenGen.GenerateTokenUsingAuthenticator(eksDetails.ClusterName, eksDetails.RoleARN)
	if err != nil {
		log.Printf("Failed to generate token using aws-iam-authenticator, falling back to custom method: %v", err)
		// Fallback to custom token generation
		token, err = tokenGen.GenerateToken(eksDetails.ClusterName, eksDetails.RoleARN)
		if err != nil {
			return nil, fmt.Errorf("failed to generate EKS token: %w", err)
		}
	}

	// Create Kubernetes config
	restConfig := &rest.Config{
		Host:        eksDetails.ClusterEndpoint,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte(eksDetails.ClusterCA),
		},
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return &Client{
		clientset:  clientset,
		config:     restConfig,
		appConfig:  cfg,
		tokenGen:   tokenGen,
		eksDetails: eksDetails,
	}, nil
}

// GetClientset returns the Kubernetes clientset
func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

// GetEKSDetails returns the EKS details
func (c *Client) GetEKSDetails() *KubeConfigEKSDetails {
	return c.eksDetails
}

// TestConnection tests the Kubernetes connection
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

// GetKubeconfigPath returns the path to the kubeconfig file (public function)
func GetKubeconfigPath() string {
	return getKubeconfigPath()
}

// getKubeconfigPath returns the path to the kubeconfig file
func getKubeconfigPath() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get user home directory: %v", err)
		return ""
	}

	return filepath.Join(homeDir, ".kube", "config")
}

// parseKubeConfigForEKS parses kubeconfig and extracts EKS-specific details
func parseKubeConfigForEKS(kubeconfigPath string) (*KubeConfigEKSDetails, error) {
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("kubeconfig path is empty")
	}

	// Load kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}

	// Get current context
	currentContext := config.CurrentContext
	if currentContext == "" {
		return nil, fmt.Errorf("no current context set in kubeconfig")
	}

	context, exists := config.Contexts[currentContext]
	if !exists {
		return nil, fmt.Errorf("current context %s not found in kubeconfig", currentContext)
	}

	// Get cluster info
	cluster, exists := config.Clusters[context.Cluster]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found in kubeconfig", context.Cluster)
	}

	// Decode CA certificate
	var clusterCA string
	if cluster.CertificateAuthorityData != nil {
		clusterCA = string(cluster.CertificateAuthorityData)
	} else if cluster.CertificateAuthority != "" {
		caData, err := os.ReadFile(cluster.CertificateAuthority)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file %s: %w", cluster.CertificateAuthority, err)
		}
		clusterCA = string(caData)
	}

	// Extract cluster name from server URL or context
	clusterName := context.Cluster
	if strings.Contains(cluster.Server, ".eks.") {
		// For EKS, use the context name as cluster name since it's more reliable
		clusterName = context.Cluster
	}

	// Extract region from server URL
	region := ""
	if strings.Contains(cluster.Server, ".eks.") {
		parts := strings.Split(cluster.Server, ".")
		for i, part := range parts {
			if part == "eks" && i+1 < len(parts) {
				region = parts[i+1]
				break
			}
		}
	}

	// Check for role ARN in user config - handle both long and short form
	roleARN := ""
	if user, exists := config.AuthInfos[context.AuthInfo]; exists {
		if user.Exec != nil {
			for i, arg := range user.Exec.Args {
				// Check for both --role-arn and -r
				if (arg == "--role-arn" || arg == "-r") && i+1 < len(user.Exec.Args) {
					roleARN = user.Exec.Args[i+1]
					break
				}
			}
		}
	}

	return &KubeConfigEKSDetails{
		ClusterName:     clusterName,
		ClusterEndpoint: cluster.Server,
		ClusterCA:       clusterCA,
		Region:          region,
		RoleARN:         roleARN,
	}, nil
}

// GetClusterCA returns the cluster CA certificate
func GetClusterCA(kubeconfigPath string) (string, error) {
	eksDetails, err := parseKubeConfigForEKS(kubeconfigPath)
	if err != nil {
		return "", err
	}
	return eksDetails.ClusterCA, nil
}
