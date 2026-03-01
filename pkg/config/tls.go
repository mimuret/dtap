package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
)

type TLSServerConfig struct {
	// Certificate Authority certificate file path.
	InterCertificates []string `hcl:"inter_certificates,optional"`
	// Certificate file path.
	Certificate string `hcl:"certificate"`
	// Private key file path.
	PrivateKey string `hcl:"private_key"`
	// ClientCACertificate specifies the CA certificate file for client authentication.
	ClientCACertificate string `hcl:"client_ca_certificate,optional"`
}

func (cfg *TLSServerConfig) Listen(hostAndPort string) (net.Listener, error) {
	if cfg.Certificate == "" || cfg.PrivateKey == "" {
		return nil, fmt.Errorf("certificate and private key must be specified")
	}

	// Load the server's certificate and private key
	certificates, err := tls.LoadX509KeyPair(cfg.Certificate, cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate and private key: %w", err)
	}

	// read intermediate certificates if provided
	for _, certFile := range cfg.InterCertificates {
		certData, err := os.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read intermediate certificate %s: %w", certFile, err)
		}
		certificates.Certificate = append(certificates.Certificate, certData)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificates},
		MinVersion:   tls.VersionTLS12,
	}

	// mTLS configuration
	if cfg.ClientCACertificate != "" {
		caCert, err := os.ReadFile(cfg.ClientCACertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to read client CA certificate: %w", err)
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caCert)
		tlsConfig.ClientCAs = certPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tls.Listen("tcp", hostAndPort, tlsConfig)
}

type TLSClientConfig struct {
	Insecure      bool   `hcl:"insecure,optional"`
	Certificate   string `hcl:"certificate,optional"`
	PrivateKey    string `hcl:"private_key,optional"`
	CACertificate string `hcl:"ca_certificate,optional"`
}

func (cfg *TLSClientConfig) Dial(hostAndPort string) (net.Conn, error) {
	return cfg.DialWithContext(context.Background(), hostAndPort)
}

func (cfg *TLSClientConfig) CryptoTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
	}

	if cfg.Certificate != "" && cfg.PrivateKey != "" {
		certificates, err := tls.LoadX509KeyPair(cfg.Certificate, cfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate and private key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificates}
	}

	if cfg.CACertificate != "" {
		caCert, err := os.ReadFile(cfg.CACertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = certPool
	}
	return tlsConfig, nil
}

func (cfg *TLSClientConfig) DialWithContext(ctx context.Context, hostAndPort string) (net.Conn, error) {
	tlsConfig, err := cfg.CryptoTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}
	dial := tls.Dialer{
		Config: tlsConfig,
	}
	return dial.DialContext(ctx, "tcp", hostAndPort)
}
