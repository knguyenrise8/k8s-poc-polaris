# Kubernetes Web Service - Certificate Analysis Tool

A Go-based web service for analyzing and monitoring Kubernetes cluster certificates, providing detailed insights into certificate expiry, pod configurations, and cluster CA information.

## 🚀 Features

### Core Functionality
- **Kubernetes Connectivity Testing** - Verify cluster connection and authentication
- **Pod Certificate Analysis** - Comprehensive analysis of certificate mounts in pods
- **Cluster CA Monitoring** - Monitor cluster CA certificate expiry with detailed date formatting
- **Certificate Expiry Tracking** - Track certificate expiry across namespaces with customizable warning thresholds
- **Debug & Diagnostics** - Built-in debugging tools for AWS and Kubernetes configuration

### API Endpoints
- `GET /` - Service overview and quick start guide
- `GET /connect-k8s` - Test Kubernetes cluster connectivity
- `GET /list-pods` - List pods in specified namespace
- `GET /cluster-ca` - Retrieve cluster CA certificate information
- `GET /cluster-ca-expiry` - Detailed cluster CA expiry analysis with human-readable dates
- `GET /pod-certificates` - Analyze certificate mounts across pods
- `GET /pod-certificates/{pod-name}` - Detailed certificate analysis for specific pod
- `GET /certificate-expiry` - Certificate expiry analysis across namespace
- `GET /debug` - Debug AWS and Kubernetes configuration
- `GET /test-k8s-auth` - Comprehensive Kubernetes authentication testing
- `GET /api-docs` - Complete API documentation with examples

## 📋 Prerequisites

- Go 1.22 or later
- AWS CLI configured or AWS credentials
- Access to an EKS cluster or Kubernetes cluster with AWS IAM authentication
- Valid kubeconfig file

## 🛠️ Installation

### 1. Clone the Repository
```bash
git clone <repository-url>
cd k8s-web-service
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Configure the Service
```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` with your configuration:
```yaml
# AWS Configuration
aws:
  access_key_id: "your-aws-access-key-id"
  secret_access_key: "your-aws-secret-access-key"
  region: "us-gov-west-1"

# Kubernetes Configuration  
kubernetes:
  cluster_name: "your-cluster-name"
  cluster_endpoint: "https://your-cluster-endpoint.eks.amazonaws.com"
  default_namespace: "default"

# Server Configuration
server:
  host: "localhost"
  port: "8080"
```

### 4. Build and Run
```bash
# Build the application
go build -o k8s-web-service ./cmd/k8s-web-service

# Run the service
./k8s-web-service
```

## 🔧 Configuration Options

### AWS Configuration
- `access_key_id` - AWS Access Key ID
- `secret_access_key` - AWS Secret Access Key  
- `region` - AWS region (e.g., us-gov-west-1, us-east-1)

### Kubernetes Configuration
- `cluster_name` - Name of your EKS/Kubernetes cluster
- `cluster_endpoint` - Kubernetes API server endpoint
- `default_namespace` - Default namespace for operations (defaults to "default")

### Server Configuration
- `host` - Server bind address (defaults to "localhost")
- `port` - Server port (defaults to "8080")

## 📖 Usage Examples

### Basic Connectivity Test
```bash
curl http://localhost:8080/connect-k8s
```

### List Pods in Namespace
```bash
# Default namespace
curl http://localhost:8080/list-pods

# Specific namespace
curl http://localhost:8080/list-pods?namespace=kube-system
```

### Certificate Analysis
```bash
# Basic pod certificate analysis
curl http://localhost:8080/pod-certificates

# Detailed analysis with expiry information
curl http://localhost:8080/pod-certificates?detailed=true&warning_days=90

# Analyze specific pod
curl http://localhost:8080/pod-certificates/my-pod-name?namespace=default&warning_days=30
```

### Cluster CA Expiry Analysis
```bash
# Default warning threshold (30 days)
curl http://localhost:8080/cluster-ca-expiry

# Custom warning threshold
curl http://localhost:8080/cluster-ca-expiry?warning_days=365
```

### Certificate Expiry Monitoring
```bash
# Monitor certificate expiry across namespace
curl http://localhost:8080/certificate-expiry?namespace=production&warning_days=60
```

## 🏗️ Project Structure

```
k8s-web-service/
├── cmd/k8s-web-service/
│   └── main.go                 # Application entry point
├── internal/
│   ├── auth/
│   │   └── aws.go             # AWS authentication utilities
│   ├── config/
│   │   └── config.go          # Configuration management
│   ├── handlers/
│   │   ├── base.go            # Handler struct and constructor
│   │   ├── types.go           # Type definitions
│   │   ├── kubernetes.go      # Basic Kubernetes operations
│   │   ├── cluster_ca.go      # Cluster CA operations
│   │   ├── pod_certificates.go # Pod certificate analysis
│   │   ├── debug.go           # Debug and utility functions
│   │   └── api_docs.go        # API documentation handler
│   └── k8s/
│       ├── client.go          # Kubernetes client management
│       └── certificates.go    # Certificate analysis utilities
├── pkg/utils/
│   └── cert.go                # Certificate utility functions
├── config.yaml.example       # Example configuration file
├── go.mod                     # Go module definition
└── README.md                  # This file
```

## 🔍 API Response Examples

### Pod Certificate Analysis (Detailed)
```json
{
  "status": "success",
  "message": "Pod certificates analyzed successfully",
  "target_namespace": "default",
  "cluster_ca_info": {
    "description": "Cluster CA Certificate",
    "length": 1099,
    "source": "cluster"
  },
  "pods": [
    {
      "name": "my-app-pod",
      "namespace": "default",
      "volume_mounts": [
        {
          "name": "service-account-token",
          "mount_path": "/var/run/secrets/kubernetes.io/serviceaccount",
          "read_only": true,
          "container": "main"
        }
      ],
      "certificate_sources": {
        "/var/run/secrets/kubernetes.io/serviceaccount": {
          "certificates": [
            {
              "subject": "CN=system:serviceaccount:default:default",
              "issuer": "CN=kubernetes",
              "not_before": "2024-01-01T00:00:00Z",
              "not_after": "2025-01-01T00:00:00Z",
              "days_until_expiry": 180
            }
          ]
        }
      },
      "expiry_warnings": []
    }
  ]
}
```

### Cluster CA Expiry Analysis
```json
{
  "status": "success",
  "analysis_date": "May 23, 2025 at 2:19 PM CDT",
  "certificate_info": {
    "enhanced_info": {
      "validity_period": {
        "not_before_formatted": "May 4, 2023 at 5:37 PM UTC",
        "not_after_formatted": "May 1, 2033 at 5:37 PM UTC",
        "valid_for_days": 3650
      },
      "expiry_info": {
        "days_until_expiry": 2899,
        "expires_on": "May 1, 2033",
        "expires_on_weekday": "Sunday, May 1, 2033",
        "time_remaining": "7 years, 344 days"
      }
    }
  },
  "summary": {
    "status_summary": "VALID (2899 days remaining)"
  }
}
```

## 🛡️ Security Considerations

- **NEVER commit AWS credentials to version control**
- **Use environment variables for credentials:**

  ```bash
  export AWS_ACCESS_KEY_ID="your-access-key"
  export AWS_SECRET_ACCESS_KEY="your-secret-key"
  ```

- **Prefer IAM roles over access keys when possible**
- Store AWS credentials securely (use IAM roles when possible)
- Limit Kubernetes permissions to read-only operations
- Use HTTPS in production environments
- Regularly rotate AWS access keys
- Monitor access logs for unauthorized usage
- Add `config.yaml` to `.gitignore` to prevent credential exposure

## 🚨 Troubleshooting

### Common Issues

1. **AWS Authentication Errors**
   - Verify AWS credentials in config.yaml
   - Check IAM permissions for EKS access
   - Use `/debug` endpoint to diagnose AWS configuration

2. **Kubernetes Connection Issues**
   - Verify cluster endpoint and name in config.yaml
   - Check kubeconfig file validity
   - Use `/test-k8s-auth` endpoint for comprehensive authentication testing

3. **Certificate Analysis Errors**
   - Ensure proper RBAC permissions to read pods and secrets
   - Verify namespace exists and is accessible
   - Check pod status (only running pods are analyzed)

### Debug Endpoints

- `GET /debug` - Check AWS and Kubernetes configuration
- `GET /test-k8s-auth` - Test authentication and permissions
- `GET /api-docs` - Complete API documentation

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Check the `/api-docs` endpoint for detailed API documentation
- Use the `/debug` and `/test-k8s-auth` endpoints for troubleshooting
- Review the troubleshooting section above
- Open an issue in the repository for bugs or feature requests