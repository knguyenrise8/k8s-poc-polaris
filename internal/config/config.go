package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	AWS struct {
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
		Region          string `yaml:"region"`
	} `yaml:"aws"`

	Kubernetes struct {
		ClusterName      string `yaml:"cluster_name"`
		ClusterEndpoint  string `yaml:"cluster_endpoint"`
		DefaultNamespace string `yaml:"default_namespace"`
	} `yaml:"kubernetes"`

	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	config := &Config{}

	// Load from YAML file if it exists
	if _, err := os.Stat(configPath); err == nil {
		file, err := os.Open(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(config); err != nil {
			return nil, fmt.Errorf("failed to decode config file: %w", err)
		}
	}

	// Override with environment variables if they exist (more secure)
	if awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID"); awsAccessKey != "" {
		config.AWS.AccessKeyID = awsAccessKey
	}
	if awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); awsSecretKey != "" {
		config.AWS.SecretAccessKey = awsSecretKey
	}
	if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
		config.AWS.Region = awsRegion
	}
	if k8sClusterName := os.Getenv("K8S_CLUSTER_NAME"); k8sClusterName != "" {
		config.Kubernetes.ClusterName = k8sClusterName
	}
	if k8sEndpoint := os.Getenv("K8S_CLUSTER_ENDPOINT"); k8sEndpoint != "" {
		config.Kubernetes.ClusterEndpoint = k8sEndpoint
	}
	if k8sDefaultNamespace := os.Getenv("K8S_DEFAULT_NAMESPACE"); k8sDefaultNamespace != "" {
		config.Kubernetes.DefaultNamespace = k8sDefaultNamespace
	}
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		config.Server.Port = serverPort
	}

	return config, nil
}

// ValidateAWSConfig checks if required AWS credentials are present
func (c *Config) ValidateAWSConfig() error {
	// Allow for no explicit AWS creds if relying on EC2 instance profile, env vars, or shared credentials
	// However, region should ideally be present for EKS.
	if c.AWS.Region == "" {
		log.Println("Warning: AWS region is not configured in config.yaml. Relying on SDK default behavior or kubeconfig.")
		// Not returning an error, as SDK might pick it up, or kubeconfig might specify it.
	}

	// If both access key and secret are empty, assume we're using alternative credential sources
	if c.AWS.AccessKeyID == "" && c.AWS.SecretAccessKey == "" {
		log.Println("Info: No explicit AWS credentials in config.yaml. Using AWS SDK default credential chain (env vars, shared credentials, instance profile, etc.)")
		return nil
	}

	// If one is set but not the other, that's an error
	if c.AWS.AccessKeyID == "" {
		return fmt.Errorf("AWS Access Key ID is required when using static credentials")
	}
	if c.AWS.SecretAccessKey == "" {
		return fmt.Errorf("AWS Secret Access Key is required when using static credentials")
	}

	return nil
}
