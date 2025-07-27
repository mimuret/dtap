/*
 * Copyright (c) 2022 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/mimuret/dtap/v3/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TLSConfig", func() {
	var (
		serverCfg                                 *config.TLSServerConfig
		clientCfg                                 *config.TLSClientConfig
		serverCertFile, serverKeyFile, caCertFile *os.File
		listener                                  net.Listener
	)

	BeforeEach(func() {
		// Generate a self-signed certificate and corresponding private key
		certPEM, keyPEM := generateSelfSignedCert()

		// Create temporary files for certificates and keys
		serverCertFile = createTempFile("server.crt", certPEM)
		serverKeyFile = createTempFile("server.key", keyPEM)
		caCertFile = createTempFile("ca.crt", certPEM)

		serverCfg = &config.TLSServerConfig{
			Certificate:         serverCertFile.Name(),
			PrivateKey:          serverKeyFile.Name(),
			ClientCACertificate: caCertFile.Name(),
		}

		clientCfg = &config.TLSClientConfig{
			CACertificate: caCertFile.Name(),
		}
	})

	AfterEach(func() {
		// Clean up temporary files
		serverCertFile.Close()
		serverKeyFile.Close()
		caCertFile.Close()
		os.Remove(serverCertFile.Name())
		os.Remove(serverKeyFile.Name())
		os.Remove(caCertFile.Name())

		if listener != nil {
			listener.Close()
		}
	})

	Context("ListenTLS", func() {
		It("should start a TLS server successfully", func() {
			listener, err := serverCfg.Listen("127.0.0.1:0")
			Expect(err).To(Succeed())
			Expect(listener).ToNot(BeNil())
			listener.Close()
		})

		It("should fail if certificate or key is missing", func() {
			serverCfg.Certificate = ""

			_, err := serverCfg.Listen("127.0.0.1:0")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("certificate and private key must be specified"))
		})
	})

	Context("DialTLS", func() {
		var (
			ln    net.Listener
			err   error
			sconn net.Conn
		)
		BeforeEach(func() {
			ln, err = serverCfg.Listen("127.0.0.1:0")
			Expect(err).To(Succeed())
			Expect(ln).ToNot(BeNil())
			go func() {
				for {
					sconn, err = ln.Accept()
					if err != nil {
						return
					}
					sconn.Write([]byte("Hello from server!"))
					sconn.Close() // Close the connection after accepting
				}
			}()
		})
		AfterEach(func() {
			if ln != nil {
				ln.Close()
			}
		})
		It("should connect to a TLS server successfully", func() {
			conn, err := clientCfg.Dial(ln.Addr().String())
			Expect(err).To(Succeed())
			Expect(conn).ToNot(BeNil())
			conn.Close()
		})
		It("should fail if CA certificate is missing", func() {
			clientCfg.CACertificate = ""
			_, err := clientCfg.Dial(ln.Addr().String())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("x509: certificate signed by unknown authority"))
		})
		It("insecurely connect to a TLS server without CA certificate", func() {
			clientCfg.CACertificate = ""
			clientCfg.Insecure = true
			conn, err := clientCfg.Dial(ln.Addr().String())
			Expect(err).To(Succeed())
			Expect(conn).ToNot(BeNil())
			conn.Close()
		})
	})
})

// createTempFile creates a temporary file with the given name and content.
func createTempFile(name, content string) *os.File {
	tmpFile, err := os.CreateTemp("", name)
	Expect(err).To(Succeed())

	_, err = tmpFile.Write([]byte(content))
	Expect(err).To(Succeed())

	return tmpFile
}

// generateSelfSignedCert generates a self-signed certificate and private key.
func generateSelfSignedCert() (certPEM, keyPEM string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).To(Succeed())

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	Expect(err).To(Succeed())

	certPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEMBytes, err := x509.MarshalECPrivateKey(priv)
	Expect(err).To(Succeed())
	keyPEMBytes = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyPEMBytes})

	return string(certPEMBytes), string(keyPEMBytes)
}
