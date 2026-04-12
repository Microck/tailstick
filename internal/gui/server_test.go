package gui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailstick/tailstick/internal/model"
)

func TestPresetsRedactsSecretsAndOnlyAllowsGet(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "tailstick.config.json")
	configBody := `{
  "defaultPreset": "ops",
  "presets": [
    {
      "id": "ops",
      "description": "Operations",
      "authKey": "tskey-auth-secret",
      "authKeyEnv": "TAILSTICK_AUTH_KEY",
      "ephemeralAuthKey": "tskey-ephemeral-secret",
      "ephemeralAuthKeyEnv": "TAILSTICK_EPHEMERAL_AUTH_KEY",
      "tags": ["tag:ops"],
      "acceptRoutes": true,
      "allowExitNodeSelection": true,
      "approvedExitNodes": ["100.64.0.1"],
      "cleanup": {
        "apiKey": "tskey-api-secret",
        "apiKeyEnv": "TAILSTICK_API_KEY",
        "deviceDeleteEnabled": true
      }
    }
  ]
}`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	srv := &Server{ConfigPath: configPath}

	req := httptest.NewRequest(http.MethodGet, "/api/presets", nil)
	rec := httptest.NewRecorder()
	srv.presets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, forbidden := range []string{
		"authKey",
		"authKeyEnv",
		"ephemeralAuthKey",
		"ephemeralAuthKeyEnv",
		"apiKey",
		"apiKeyEnv",
		"tskey-auth-secret",
		"tskey-api-secret",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `"id":"ops"`) {
		t.Fatalf("expected preset id in response, got %s", body)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/presets", nil)
	rec = httptest.NewRecorder()
	srv.presets(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got status %d want 405", rec.Code)
	}
}

func TestEnrollRejectsInvalidModeAndNegativeDurations(t *testing.T) {
	srv := &Server{
		EnrollFn: func(context.Context, model.RuntimeOptions) (model.LeaseRecord, error) {
			t.Fatal("enroll should not be called for invalid input")
			return model.LeaseRecord{}, nil
		},
	}

	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "invalid mode",
			body: `{"mode":"bogus","channel":"stable"}`,
			want: `invalid mode "bogus"`,
		},
		{
			name: "invalid channel",
			body: `{"mode":"timed","channel":"bogus"}`,
			want: `invalid channel "bogus"`,
		},
		{
			name: "negative days",
			body: `{"mode":"timed","channel":"stable","days":-1}`,
			want: `days must be non-negative`,
		},
		{
			name: "negative custom days",
			body: `{"mode":"timed","channel":"stable","customDays":-1}`,
			want: `customDays must be non-negative`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/enroll", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			srv.enroll(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got status %d want 400", rec.Code)
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got := payload["error"]; got == nil || !strings.Contains(got.(string), tc.want) {
				t.Fatalf("got error %v want substring %q", got, tc.want)
			}
		})
	}
}
