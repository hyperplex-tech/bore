package tailscale

import (
	"net"
	"testing"
)

func TestIsTailscaleIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"100.64.0.1", true},     // Start of CGNAT range
		{"100.127.255.254", true}, // End of CGNAT range
		{"100.100.50.50", true},   // Typical Tailscale IP
		{"100.63.255.255", false}, // Just below range
		{"100.128.0.0", false},    // Just above range
		{"192.168.1.1", false},    // Private, not Tailscale
		{"10.0.0.1", false},       // Private, not Tailscale
		{"8.8.8.8", false},        // Public
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := isTailscaleIP(ip)
		if got != tt.want {
			t.Errorf("isTailscaleIP(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIsTailscaleAddrMagicDNS(t *testing.T) {
	// .ts.net suffix is always Tailscale.
	if !IsTailscaleAddr("myhost.tail1234.ts.net") {
		t.Error("expected .ts.net to be Tailscale")
	}
	if !IsTailscaleAddr("server.example.ts.net") {
		t.Error("expected .ts.net suffix to be Tailscale")
	}
}

func TestIsTailscaleAddrNonTailscale(t *testing.T) {
	// Regular hosts should not be Tailscale (unless they resolve to CGNAT).
	if IsTailscaleAddr("google.com") {
		t.Error("google.com should not be Tailscale")
	}
	if IsTailscaleAddr("192.168.1.1") {
		t.Error("192.168.1.1 should not be Tailscale")
	}
}

func TestDetectReturnsStruct(t *testing.T) {
	// Just verify Detect() doesn't panic and returns a valid struct.
	status := Detect()
	// On machines without tailscale, Available will be false.
	_ = status.Available
	_ = status.Running
	_ = status.IP
}
