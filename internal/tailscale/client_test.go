package tailscale

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDeleteDeviceTreatsNotFoundAsAlreadyDeleted(t *testing.T) {
	originalClient := deleteDeviceHTTPClient
	deleteDeviceHTTPClient = &http.Client{
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
	t.Cleanup(func() {
		deleteDeviceHTTPClient = originalClient
	})

	if err := DeleteDevice(context.Background(), "tskey-api-example", "device-123"); err != nil {
		t.Fatalf("expected 404 delete to be treated as success, got %v", err)
	}
}

func TestDeleteDeviceReturnsErrorForOtherFailures(t *testing.T) {
	originalClient := deleteDeviceHTTPClient
	deleteDeviceHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"message":"forbidden"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		deleteDeviceHTTPClient = originalClient
	})

	err := DeleteDevice(context.Background(), "tskey-api-example", "device-123")
	if err == nil {
		t.Fatal("expected delete error")
	}
	if !strings.Contains(err.Error(), "status=403") {
		t.Fatalf("got error %q want 403 context", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
