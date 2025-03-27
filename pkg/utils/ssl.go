package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

// GenerateSelfSignedCert generates a self-signed certificate for development
// Returns paths to the certificate and key files
func GenerateSelfSignedCert(hostname string) (certPath, keyPath string, err error) {
	// Create a temporary directory for the certificate and key files
	tempDir := os.TempDir()
	certPath = tempDir + "/server.crt"
	keyPath = tempDir + "/server.key"

	log.Printf("Generating self-signed certificate in %s for host %s", tempDir, hostname)

	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Create a certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year validity

	// Extract domain from hostname (remove protocol and port)
	domain := hostname
	if strings.Contains(domain, "://") {
		domain = strings.Split(domain, "://")[1]
	}
	if strings.Contains(domain, ":") {
		domain = strings.Split(domain, ":")[0]
	}

	// Default to localhost if no valid domain
	if domain == "" {
		domain = "localhost"
	}

	// Prepare DNS names
	dnsNames := []string{domain}
	if domain != "localhost" {
		dnsNames = append(dnsNames, "localhost")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Slack Kudos Dev"},
			CommonName:   domain,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create a self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	// Write the certificate to a file
	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		certOut.Close()
		return "", "", err
	}
	certOut.Close()

	// Write the private key to a file
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return "", "", err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		keyOut.Close()
		return "", "", err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		keyOut.Close()
		return "", "", err
	}
	keyOut.Close()

	log.Printf("Self-signed certificate generated: %s, %s", certPath, keyPath)
	return certPath, keyPath, nil
}
