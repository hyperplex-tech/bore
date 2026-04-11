package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// CertProvider authenticates using an SSH certificate + private key pair.
// The certificate file is expected at <identity_file>-cert.pub.
type CertProvider struct{}

func (p *CertProvider) AuthMethods(cfg AuthConfig) ([]ssh.AuthMethod, error) {
	keyPath := cfg.IdentityFile
	if keyPath == "" {
		return nil, fmt.Errorf("cert auth requires an identity file")
	}

	// Expand ~.
	if strings.HasPrefix(keyPath, "~/") {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[2:])
	}

	// Read private key.
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing key %s: %w", keyPath, err)
	}

	// Read certificate.
	certPath := keyPath + "-cert.pub"
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("reading certificate %s: %w", certPath, err)
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(certData)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate %s: %w", certPath, err)
	}

	cert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("%s is not an SSH certificate", certPath)
	}

	certSigner, err := ssh.NewCertSigner(cert, signer)
	if err != nil {
		return nil, fmt.Errorf("creating cert signer: %w", err)
	}

	return []ssh.AuthMethod{ssh.PublicKeys(certSigner)}, nil
}
