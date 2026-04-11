package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	err := Run("true", Env{TunnelName: "test", Status: "connecting"})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestRunFailure(t *testing.T) {
	err := Run("false", Env{TunnelName: "test"})
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
}

func TestRunEmpty(t *testing.T) {
	err := Run("", Env{TunnelName: "test"})
	if err != nil {
		t.Fatalf("empty command should be no-op, got: %v", err)
	}
}

func TestRunEnvironmentVariables(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "env.txt")

	cmd := `env | grep BORE_ > ` + outFile
	err := Run(cmd, Env{
		TunnelName: "my-tunnel",
		Group:      "dev",
		LocalHost:  "127.0.0.1",
		LocalPort:  5432,
		RemoteHost: "db.internal",
		RemotePort: 5432,
		SSHHost:    "bastion.example.com",
		SSHPort:    22,
		SSHUser:    "deploy",
		Status:     "connecting",
	})
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	output := string(data)
	expected := []string{
		"BORE_TUNNEL_NAME=my-tunnel",
		"BORE_GROUP=dev",
		"BORE_LOCAL_PORT=5432",
		"BORE_REMOTE_HOST=db.internal",
		"BORE_SSH_HOST=bastion.example.com",
		"BORE_STATUS=connecting",
	}
	for _, e := range expected {
		if !contains(output, e) {
			t.Errorf("missing env var %q in output:\n%s", e, output)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
