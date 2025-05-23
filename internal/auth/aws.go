package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	appConfig "k8s-web-service/internal/config"
)

// EKSTokenGenerator handles EKS token generation
type EKSTokenGenerator struct {
	cfg *appConfig.Config
}

// NewEKSTokenGenerator creates a new EKS token generator
func NewEKSTokenGenerator(cfg *appConfig.Config) *EKSTokenGenerator {
	return &EKSTokenGenerator{cfg: cfg}
}

// GenerateToken generates an EKS authentication token
func (e *EKSTokenGenerator) GenerateToken(clusterName string, roleARNToAssume string) (string, error) {
	ctx := context.Background()

	// Load AWS configuration
	var awsCfg aws.Config
	var err error

	if e.cfg.AWS.AccessKeyID != "" && e.cfg.AWS.SecretAccessKey != "" {
		// Use static credentials from config
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				e.cfg.AWS.AccessKeyID,
				e.cfg.AWS.SecretAccessKey,
				"",
			)),
			config.WithRegion(e.cfg.AWS.Region),
		)
	} else {
		// Use default credential chain (env vars, shared credentials, instance profile, etc.)
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(e.cfg.AWS.Region),
		)
	}

	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If a role ARN is provided, assume the role
	if roleARNToAssume != "" {
		log.Printf("Attempting to assume role: %s", roleARNToAssume)
		stsClient := sts.NewFromConfig(awsCfg)

		assumeRoleInput := &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleARNToAssume),
			RoleSessionName: aws.String("k8s-web-service-session"),
		}

		assumeRoleOutput, err := stsClient.AssumeRole(ctx, assumeRoleInput)
		if err != nil {
			log.Printf("Failed to assume role %s: %v", roleARNToAssume, err)
			return "", fmt.Errorf("failed to assume role %s: %w", roleARNToAssume, err)
		}

		log.Printf("Successfully assumed role: %s", roleARNToAssume)

		// Update AWS config with assumed role credentials
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(
			*assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken,
		)
	}

	// Verify credentials work
	stsClient := sts.NewFromConfig(awsCfg)
	callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	log.Printf("AWS Caller Identity - Account: %s, ARN: %s, UserID: %s",
		*callerIdentity.Account, *callerIdentity.Arn, *callerIdentity.UserId)

	// Create presigned URL for GetCallerIdentity with cluster name header
	presignClient := sts.NewPresignClient(stsClient)

	presignedURL, err := presignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to presign GetCallerIdentity: %w", err)
	}

	// Extract the URL and add the cluster name header
	urlStr := presignedURL.URL

	// Add the x-k8s-aws-id header to the URL as a query parameter
	// This is how aws-iam-authenticator includes the cluster name
	if !strings.Contains(urlStr, "X-K8s-Aws-Id") {
		separator := "&"
		if !strings.Contains(urlStr, "?") {
			separator = "?"
		}
		urlStr = fmt.Sprintf("%s%sX-K8s-Aws-Id=%s", urlStr, separator, clusterName)
	}

	// Create the token payload
	tokenPayload := fmt.Sprintf("k8s-aws-v1.%s", base64.RawURLEncoding.EncodeToString([]byte(urlStr)))

	return tokenPayload, nil
}

// GenerateTokenUsingAuthenticator generates an EKS token using aws-iam-authenticator directly
func (e *EKSTokenGenerator) GenerateTokenUsingAuthenticator(clusterName string, roleARN string) (string, error) {
	// Build the command arguments
	args := []string{"token", "-i", clusterName}
	if roleARN != "" {
		args = append(args, "-r", roleARN)
	}

	// Execute aws-iam-authenticator
	cmd := exec.Command("aws-iam-authenticator", args...)

	// Inherit the current environment to use the same AWS configuration as kubectl
	cmd.Env = os.Environ()

	// Override with our specific AWS credentials if they exist
	if e.cfg.AWS.AccessKeyID != "" && e.cfg.AWS.SecretAccessKey != "" {
		// Find and replace or add AWS environment variables
		envMap := make(map[string]string)
		for _, env := range cmd.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		envMap["AWS_ACCESS_KEY_ID"] = e.cfg.AWS.AccessKeyID
		envMap["AWS_SECRET_ACCESS_KEY"] = e.cfg.AWS.SecretAccessKey

		if e.cfg.AWS.Region != "" {
			envMap["AWS_REGION"] = e.cfg.AWS.Region
		}

		// Rebuild environment slice
		cmd.Env = []string{}
		for key, value := range envMap {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute aws-iam-authenticator: %w", err)
	}

	// Parse the JSON output
	var execCredential struct {
		Status struct {
			Token string `json:"token"`
		} `json:"status"`
	}

	if err := json.Unmarshal(output, &execCredential); err != nil {
		return "", fmt.Errorf("failed to parse aws-iam-authenticator output: %w", err)
	}

	return execCredential.Status.Token, nil
}

// GetCallerIdentity returns the AWS caller identity for debugging
func (e *EKSTokenGenerator) GetCallerIdentity() (*sts.GetCallerIdentityOutput, error) {
	ctx := context.Background()

	// Load AWS configuration
	var awsCfg aws.Config
	var err error

	if e.cfg.AWS.AccessKeyID != "" && e.cfg.AWS.SecretAccessKey != "" {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				e.cfg.AWS.AccessKeyID,
				e.cfg.AWS.SecretAccessKey,
				"",
			)),
			config.WithRegion(e.cfg.AWS.Region),
		)
	} else {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(e.cfg.AWS.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(awsCfg)
	return stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}
