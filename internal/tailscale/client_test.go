package tailscale

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailstick/tailstick/internal/model"
	"github.com/tailstick/tailstick/internal/platform"
)

func TestDeleteDeviceTreatsNotFoundAsAlreadyDeleted(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodDelete {
				t.Fatalf("got method %s want DELETE", req.Method)
			}
			if req.URL.String() != "https://api.tailscale.com/api/v2/device/device-123" {
				t.Fatalf("got URL %s", req.URL.String())
			}
			user, pass, ok := req.BasicAuth()
			if !ok || user != "tskey-api-example" || pass != "" {
				t.Fatalf("unexpected basic auth user=%q pass=%q ok=%v", user, pass, ok)
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"message":"no manageable device matching this ID found"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	if err := deleteDevice(context.Background(), client, "tskey-api-example", "device-123"); err != nil {
		t.Fatalf("expected 404 delete to be treated as success, got %v", err)
	}
}

func TestDeleteDeviceReturnsErrorForOtherFailures(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"message":"forbidden"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := deleteDevice(context.Background(), client, "tskey-api-example", "device-123")
	if err == nil {
		t.Fatal("expected delete error")
	}
	if !strings.Contains(err.Error(), "status=403") {
		t.Fatalf("got error %q want 403 context", err)
	}
}

func TestDefaultDeleteDeviceClientHasTimeout(t *testing.T) {
	if defaultDeleteDeviceHTTPClient.Timeout <= 0 {
		t.Fatalf("expected default delete client timeout, got %s", defaultDeleteDeviceHTTPClient.Timeout)
	}
}

func TestClientUpUsesAuthKeyFileAndRemovesIt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem runner test in short mode")
	}

	root := t.TempDir()
	logPath := filepath.Join(root, "tailscale.log")
	scriptPath := filepath.Join(root, "tailscale")
	script := `#!/bin/sh
set -eu
for arg in "$@"; do
  printf '%s\n' "$arg" >> "` + logPath + `"
  case "$arg" in
    --auth-key=file:*)
      key_path=${arg#--auth-key=file:}
      printf 'key=%s\n' "$(cat "$key_path")" >> "` + logPath + `"
      ;;
  esac
done
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tailscale: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	client := Client{Runner: platform.Runner{}}
	preset := model.Preset{AuthKey: "tskey-auth-secret"}

	if err := client.Up(context.Background(), preset, "device-name", model.LeaseModeTimed, ""); err != nil {
		t.Fatalf("up: %v", err)
	}

	bodyBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	body := string(bodyBytes)
	if strings.Contains(body, "--auth-key=tskey-auth-secret") {
		t.Fatalf("raw auth key leaked into argv log: %q", body)
	}
	if !strings.Contains(body, "--auth-key=file:") {
		t.Fatalf("expected file-based auth key flag, got %q", body)
	}
	if !strings.Contains(body, "key=tskey-auth-secret") {
		t.Fatalf("expected helper script to read auth key file, got %q", body)
	}

	var keyPath string
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		if strings.HasPrefix(line, "--auth-key=file:") {
			keyPath = strings.TrimPrefix(line, "--auth-key=file:")
			break
		}
	}
	if keyPath == "" {
		t.Fatal("failed to capture auth key temp path")
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("expected auth key temp file to be removed, stat err=%v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
