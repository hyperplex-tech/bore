package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// KeyProvider authenticates using a private key file.
type KeyProvider struct{}

func (p *KeyProvider) AuthMethods(cfg AuthConfig) ([]ssh.AuthMethod, error) {
	keyPath := cfg.IdentityFile
	if keyPath == "" {
		// Try common default key paths.
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home dir: %w", err)
		}
		defaults := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_ecdsa"),
		}
		for _, path := range defaults {
			if _, err := os.Stat(path); err == nil {
				keyPath = path
				break
			}
		}
		if keyPath == "" {
			return nil, fmt.Errorf("no identity file found")
		}
	}

	// Expand ~ in path.
	if strings.HasPrefix(keyPath, "~/") {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[2:])
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parsing key %s: %w", keyPath, err)
	}

	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}
