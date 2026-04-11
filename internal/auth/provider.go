package auth

import (
	"golang.org/x/crypto/ssh"
)

// Provider resolves SSH authentication methods for a tunnel connection.
type Provider interface {
	// AuthMethods returns the SSH auth methods to use for the given config.
	AuthMethods(cfg AuthConfig) ([]ssh.AuthMethod, error)
}

// AuthConfig contains the auth-related fields needed to resolve auth methods.
type AuthConfig struct {
	Method       string // "agent", "key", "cert"
	IdentityFile string
	SSHUser      string
}

// NewProvider creates the appropriate auth provider based on method.
func NewProvider(method string) Provider {
	switch method {
	case "key":
		return &KeyProvider{}
	case "cert":
		return &CertProvider{}
	case "agent":
		fallthrough
	default:
		// Try agent first, fall back to key files (matches ssh behavior).
		return &CompositeProvider{
			providers: []Provider{&AgentProvider{}, &KeyProvider{}},
		}
	}
}

// CompositeProvider tries multiple providers and collects all auth methods.
type CompositeProvider struct {
	providers []Provider
}

func (p *CompositeProvider) AuthMethods(cfg AuthConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	for _, provider := range p.providers {
		m, err := provider.AuthMethods(cfg)
		if err != nil {
			continue // Skip providers that fail.
		}
		methods = append(methods, m...)
	}
	if len(methods) == 0 {
		return nil, ErrNoAuthMethods
	}
	return methods, nil
}
