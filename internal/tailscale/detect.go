package tailscale

import (
	"encoding/json"
	"net"
	"os/exec"
	"strings"
)

// Status represents the relevant Tailscale state.
type Status struct {
	Available bool   `json:"available"`
	Running   bool   `json:"running"`
	IP        string `json:"ip"`        // Tailscale IP of this machine
	Hostname  string `json:"hostname"`  // MagicDNS hostname
	Tailnet   string `json:"tailnet"`   // Tailnet name
	Version   string `json:"version"`
}

// Detect checks if Tailscale is installed and running.
func Detect() Status {
	s := Status{}

	// Check if tailscale CLI exists.
	path, err := exec.LookPath("tailscale")
	if err != nil {
		return s
	}
	s.Available = true

	// Get status JSON.
	out, err := exec.Command(path, "status", "--json").Output()
	if err != nil {
		return s
	}

	var tsStatus struct {
		BackendState string `json:"BackendState"`
		Self         struct {
			HostName     string   `json:"HostName"`
			DNSName      string   `json:"DNSName"`
			TailscaleIPs []string `json:"TailscaleIPs"`
		} `json:"Self"`
		CurrentTailnet struct {
			Name string `json:"Name"`
		} `json:"CurrentTailnet"`
		Version string `json:"Version"`
	}

	if err := json.Unmarshal(out, &tsStatus); err != nil {
		return s
	}

	s.Running = tsStatus.BackendState == "Running"
	s.Version = tsStatus.Version
	s.Hostname = tsStatus.Self.HostName
	s.Tailnet = tsStatus.CurrentTailnet.Name

	if len(tsStatus.Self.TailscaleIPs) > 0 {
		s.IP = tsStatus.Self.TailscaleIPs[0]
	}

	return s
}

// IsTailscaleAddr returns true if the given host resolves to a Tailscale IP
// (100.x.y.z range) or is a MagicDNS name (.ts.net suffix).
func IsTailscaleAddr(host string) bool {
	// Check for MagicDNS suffix.
	if strings.HasSuffix(host, ".ts.net") {
		return true
	}

	// Check if the host resolves to a Tailscale CGNAT range (100.64.0.0/10).
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if isTailscaleIP(ip) {
			return true
		}
	}
	return false
}

// isTailscaleIP checks if an IP is in the Tailscale CGNAT range 100.64.0.0/10.
func isTailscaleIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	// 100.64.0.0/10 = first byte 100, second byte 64-127.
	return ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127
}
