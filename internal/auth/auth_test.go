package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestKeyProviderWithRealKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")

	// Generate a test ed25519 key.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)

	provider := &KeyProvider{}
	methods, err := provider.AuthMethods(AuthConfig{
		Method:       "key",
		IdentityFile: keyPath,
	})
	if err != nil {
		t.Fatalf("KeyProvider.AuthMethods: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestKeyProviderMissingFile(t *testing.T) {
	provider := &KeyProvider{}
	_, err := provider.AuthMethods(AuthConfig{
		Method:       "key",
		IdentityFile: "/nonexistent/key",
	})
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestKeyProviderAutoDiscovery(t *testing.T) {
	// This tests that the key provider tries default paths.
	// It won't find any in a test environment (no ~/.ssh), so it should error.
	provider := &KeyProvider{}
	_, err := provider.AuthMethods(AuthConfig{Method: "key"})
	// May or may not find a key depending on the test machine,
	// but should not panic either way.
	_ = err
}

func TestCertProviderWithRealCert(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	certPath := keyPath + "-cert.pub"

	// Generate key pair.
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)

	// Create a self-signed SSH certificate.
	signer, _ := ssh.NewSignerFromKey(priv)
	sshPub, _ := ssh.NewPublicKey(pub)
	cert := &ssh.Certificate{
		Key:             sshPub,
		CertType:        ssh.UserCert,
		KeyId:           "test-cert",
		ValidPrincipals: []string{"testuser"},
		ValidAfter:      0,
		ValidBefore:     ssh.CertTimeInfinity,
	}
	cert.SignCert(rand.Reader, signer)

	// Write the certificate.
	certBytes := ssh.MarshalAuthorizedKey(cert)
	os.WriteFile(certPath, certBytes, 0o644)

	provider := &CertProvider{}
	methods, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: keyPath,
	})
	if err != nil {
		t.Fatalf("CertProvider.AuthMethods: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestCertProviderMissingCert(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)

	provider := &CertProvider{}
	_, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: keyPath,
	})
	if err == nil {
		t.Fatal("expected error for missing cert file")
	}
}

func TestCertProviderNoIdentityFile(t *testing.T) {
	provider := &CertProvider{}
	_, err := provider.AuthMethods(AuthConfig{Method: "cert"})
	if err == nil {
		t.Fatal("expected error for empty identity file")
	}
}

func TestCompositeProviderFallback(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)

	// Composite: agent (may fail) + key (should succeed).
	provider := &CompositeProvider{
		providers: []Provider{&AgentProvider{}, &KeyProvider{}},
	}
	methods, err := provider.AuthMethods(AuthConfig{
		IdentityFile: keyPath,
	})
	if err != nil {
		t.Fatalf("CompositeProvider: %v", err)
	}
	if len(methods) == 0 {
		t.Fatal("expected at least 1 auth method from composite")
	}
}

func TestCertProviderNotACertificate(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	certPath := keyPath + "-cert.pub"

	// Generate key pair.
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)

	// Write a regular public key (NOT a certificate) to the cert path.
	sshPub, _ := ssh.NewPublicKey(pub)
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)
	os.WriteFile(certPath, pubBytes, 0o644)

	provider := &CertProvider{}
	_, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: keyPath,
	})
	if err == nil {
		t.Fatal("expected error when cert file is a regular public key")
	}
	if !strings.Contains(err.Error(), "not an SSH certificate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCertProviderTildeExpansion(t *testing.T) {
	// Use a tilde path that won't exist — verify the error references the expanded path.
	provider := &CertProvider{}
	_, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: "~/nonexistent_bore_test_key",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
	home, _ := os.UserHomeDir()
	expanded := filepath.Join(home, "nonexistent_bore_test_key")
	if !strings.Contains(err.Error(), expanded) {
		t.Fatalf("error should reference expanded path %q, got: %v", expanded, err)
	}
}

func TestCertProviderCorruptKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_bad")
	os.WriteFile(keyPath, []byte("not a valid private key"), 0o600)

	provider := &CertProvider{}
	_, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: keyPath,
	})
	if err == nil {
		t.Fatal("expected error for corrupt key")
	}
	if !strings.Contains(err.Error(), "parsing key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCertProviderCorruptCertFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	certPath := keyPath + "-cert.pub"

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)

	// Write garbage to the cert file.
	os.WriteFile(certPath, []byte("not a valid certificate"), 0o644)

	provider := &CertProvider{}
	_, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: keyPath,
	})
	if err == nil {
		t.Fatal("expected error for corrupt cert file")
	}
	if !strings.Contains(err.Error(), "parsing certificate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCertProviderEndToEndSSH(t *testing.T) {
	// Full integration: create a CA, sign a user cert, set up an SSH server
	// that trusts the CA, and verify that cert auth completes a handshake.

	// 1. Generate CA key (the certificate authority).
	_, caPriv, _ := ed25519.GenerateKey(rand.Reader)
	caSigner, _ := ssh.NewSignerFromKey(caPriv)

	// 2. Generate user key pair.
	userPub, userPriv, _ := ed25519.GenerateKey(rand.Reader)
	sshUserPub, _ := ssh.NewPublicKey(userPub)

	// 3. Create a certificate signed by the CA.
	cert := &ssh.Certificate{
		Key:             sshUserPub,
		CertType:        ssh.UserCert,
		KeyId:           "test-user-cert",
		ValidPrincipals: []string{"testuser"},
		ValidAfter:      0,
		ValidBefore:     ssh.CertTimeInfinity,
	}
	cert.SignCert(rand.Reader, caSigner)

	// 4. Write key and cert to temp files.
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	block, _ := ssh.MarshalPrivateKey(userPriv, "")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600)
	os.WriteFile(keyPath+"-cert.pub", ssh.MarshalAuthorizedKey(cert), 0o644)

	// 5. Get the CertSigner from our provider.
	provider := &CertProvider{}
	methods, err := provider.AuthMethods(AuthConfig{
		Method:       "cert",
		IdentityFile: keyPath,
	})
	if err != nil {
		t.Fatalf("AuthMethods: %v", err)
	}

	// 6. Set up an SSH server that trusts the CA.
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Accept certificates signed by our CA.
			certKey, ok := key.(*ssh.Certificate)
			if !ok {
				return nil, fmt.Errorf("not a certificate")
			}
			// Verify the cert was signed by our CA.
			if !bytes.Equal(certKey.SignatureKey.Marshal(), caSigner.PublicKey().Marshal()) {
				return nil, fmt.Errorf("unknown CA")
			}
			// Verify the principal.
			if conn.User() != "testuser" {
				return nil, fmt.Errorf("wrong user")
			}
			for _, p := range certKey.ValidPrincipals {
				if p == conn.User() {
					return nil, nil
				}
			}
			return nil, fmt.Errorf("principal not allowed")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	handshakeDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			handshakeDone <- err
			return
		}
		defer conn.Close()
		_, _, _, err = ssh.NewServerConn(conn, serverConfig)
		handshakeDone <- err
	}()

	// 7. Dial the server using the cert auth method.
	clientConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            methods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", listener.Addr().String(), clientConfig)
	if err != nil {
		t.Fatalf("SSH dial with cert auth failed: %v", err)
	}
	client.Close()

	// Verify server side accepted the handshake.
	select {
	case err := <-handshakeDone:
		if err != nil {
			t.Fatalf("server handshake error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server handshake")
	}
}

func TestNewProviderFactory(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"key", "*auth.KeyProvider"},
		{"agent", "*auth.AgentProvider"},
		{"cert", "*auth.CertProvider"},
		{"", "*auth.CompositeProvider"},
		{"auto", "*auth.CompositeProvider"},
	}
	for _, tt := range tests {
		p := NewProvider(tt.method)
		got := typeString(p)
		if got != tt.want {
			t.Errorf("NewProvider(%q) = %s, want %s", tt.method, got, tt.want)
		}
	}
}

func typeString(p Provider) string {
	switch p.(type) {
	case *KeyProvider:
		return "*auth.KeyProvider"
	case *AgentProvider:
		return "*auth.AgentProvider"
	case *CertProvider:
		return "*auth.CertProvider"
	case *CompositeProvider:
		return "*auth.CompositeProvider"
	default:
		return "unknown"
	}
}
