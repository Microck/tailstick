package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvesOperatorPasswordEnv(t *testing.T) {
	t.Setenv("TAILSTICK_OPERATOR_PASSWORD", "secret-pass")

	dir := t.TempDir()
	path := filepath.Join(dir, "tailstick.config.json")
	content := `{
  "stableVersion": "1.76.6",
  "defaultPreset": "ops",
  "operatorPasswordEnv": "TAILSTICK_OPERATOR_PASSWORD",
  "presets": [
    {
      "id": "ops",
      "description": "test",
      "authKey": "tskey-auth-test",
      "tags": []
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.OperatorPassword != "secret-pass" {
		t.Fatalf("operator password env not resolved, got %q", cfg.OperatorPassword)
	}
}
