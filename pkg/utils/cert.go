package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// CertificateInfo contains parsed certificate information
type CertificateInfo struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serial_number"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	IsExpired    bool      `json:"is_expired"`
	DaysUntilExp int       `json:"days_until_expiry"`
	DNSNames     []string  `json:"dns_names,omitempty"`
	IPAddresses  []string  `json:"ip_addresses,omitempty"`
	KeyUsage     []string  `json:"key_usage,omitempty"`
	IsCA         bool      `json:"is_ca"`
}

// ParseCertificate parses a PEM-encoded certificate and extracts information
func ParseCertificate(certPEM string) (*CertificateInfo, error) {
	// Clean up the certificate string
	certPEM = strings.TrimSpace(certPEM)

	// Decode PEM block
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a certificate, found: %s", block.Type)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Calculate days until expiry
	now := time.Now()
	daysUntilExp := int(cert.NotAfter.Sub(now).Hours() / 24)
	isExpired := now.After(cert.NotAfter)

	// Extract IP addresses
	var ipAddresses []string
	for _, ip := range cert.IPAddresses {
		ipAddresses = append(ipAddresses, ip.String())
	}

	// Extract key usage
	var keyUsage []string
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		keyUsage = append(keyUsage, "Digital Signature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		keyUsage = append(keyUsage, "Key Encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
		keyUsage = append(keyUsage, "Data Encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
		keyUsage = append(keyUsage, "Key Agreement")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		keyUsage = append(keyUsage, "Certificate Sign")
	}
	if cert.KeyUsage&x509.KeyUsageCRLSign != 0 {
		keyUsage = append(keyUsage, "CRL Sign")
	}

	return &CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		IsExpired:    isExpired,
		DaysUntilExp: daysUntilExp,
		DNSNames:     cert.DNSNames,
		IPAddresses:  ipAddresses,
		KeyUsage:     keyUsage,
		IsCA:         cert.IsCA,
	}, nil
}

// ParseCertificateBundle parses multiple certificates from a bundle
func ParseCertificateBundle(certBundle string) ([]*CertificateInfo, error) {
	var certificates []*CertificateInfo

	// Split the bundle into individual certificates
	certBundle = strings.TrimSpace(certBundle)

	// Find all certificate blocks
	remaining := certBundle
	for {
		block, rest := pem.Decode([]byte(remaining))
		if block == nil {
			break
		}

		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				// Skip invalid certificates but continue processing
				remaining = string(rest)
				continue
			}

			// Calculate days until expiry
			now := time.Now()
			daysUntilExp := int(cert.NotAfter.Sub(now).Hours() / 24)
			isExpired := now.After(cert.NotAfter)

			// Extract IP addresses
			var ipAddresses []string
			for _, ip := range cert.IPAddresses {
				ipAddresses = append(ipAddresses, ip.String())
			}

			// Extract key usage
			var keyUsage []string
			if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
				keyUsage = append(keyUsage, "Digital Signature")
			}
			if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
				keyUsage = append(keyUsage, "Key Encipherment")
			}
			if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
				keyUsage = append(keyUsage, "Data Encipherment")
			}
			if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
				keyUsage = append(keyUsage, "Key Agreement")
			}
			if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
				keyUsage = append(keyUsage, "Certificate Sign")
			}
			if cert.KeyUsage&x509.KeyUsageCRLSign != 0 {
				keyUsage = append(keyUsage, "CRL Sign")
			}

			certInfo := &CertificateInfo{
				Subject:      cert.Subject.String(),
				Issuer:       cert.Issuer.String(),
				SerialNumber: cert.SerialNumber.String(),
				NotBefore:    cert.NotBefore,
				NotAfter:     cert.NotAfter,
				IsExpired:    isExpired,
				DaysUntilExp: daysUntilExp,
				DNSNames:     cert.DNSNames,
				IPAddresses:  ipAddresses,
				KeyUsage:     keyUsage,
				IsCA:         cert.IsCA,
			}

			certificates = append(certificates, certInfo)
		}

		remaining = string(rest)
		if len(remaining) == 0 {
			break
		}
	}

	if len(certificates) == 0 {
		return nil, fmt.Errorf("no valid certificates found in bundle")
	}

	return certificates, nil
}

// ValidateCertificateExpiry checks if certificates are expiring soon
func ValidateCertificateExpiry(certs []*CertificateInfo, warningDays int) []string {
	var warnings []string

	for _, cert := range certs {
		if cert.IsExpired {
			warnings = append(warnings, fmt.Sprintf("Certificate '%s' has EXPIRED on %s",
				cert.Subject, cert.NotAfter.Format("2006-01-02")))
		} else if cert.DaysUntilExp <= warningDays {
			warnings = append(warnings, fmt.Sprintf("Certificate '%s' expires in %d days (%s)",
				cert.Subject, cert.DaysUntilExp, cert.NotAfter.Format("2006-01-02")))
		}
	}

	return warnings
}
