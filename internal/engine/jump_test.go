package engine

import (
	"testing"
)

func TestNormalizeAddr(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"host.example.com", "host.example.com:22"},
		{"host.example.com:2222", "host.example.com:2222"},
		{"10.0.0.1", "10.0.0.1:22"},
		{"10.0.0.1:22", "10.0.0.1:22"},
	}
	for _, tt := range tests {
		got := normalizeAddr(tt.input)
		if got != tt.want {
			t.Errorf("normalizeAddr(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseUserHost(t *testing.T) {
	tests := []struct {
		addr, defaultUser string
		wantUser, wantHost string
	}{
		{"host.example.com:22", "default", "default", "host.example.com:22"},
		{"deploy@bastion.example.com:22", "default", "deploy", "bastion.example.com:22"},
		{"root@10.0.0.1:2222", "default", "root", "10.0.0.1:2222"},
	}
	for _, tt := range tests {
		user, host := parseUserHost(tt.addr, tt.defaultUser)
		if user != tt.wantUser || host != tt.wantHost {
			t.Errorf("parseUserHost(%q, %q) = (%q, %q), want (%q, %q)",
				tt.addr, tt.defaultUser, user, host, tt.wantUser, tt.wantHost)
		}
	}
}
