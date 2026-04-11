// Package config handles loading, validating, and resolving tailstick configuration files.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailstick/tailstick/internal/model"
)

const (
	DefaultConfigFile = "tailstick.config.json"
)

// Load reads and parses a tailstick configuration file, expanding environment variables
// and validating the result.
func Load(path string) (model.Config, error) {
	if path == "" {
		path = DefaultConfigFile
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, fmt.Errorf("read config: %w", err)
	}
	// Allow ${ENV_VAR} expansion in config values.
	expanded := os.ExpandEnv(string(b))

	var cfg model.Config
	if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
		return model.Config{}, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(cfg.OperatorPassword) == "" && strings.TrimSpace(cfg.OperatorPasswordEnv) != "" {
		cfg.OperatorPassword = strings.TrimSpace(os.Getenv(cfg.OperatorPasswordEnv))
	}
	if err := Validate(cfg); err != nil {
		return model.Config{}, err
	}
	return cfg, nil
}

// Validate checks that the configuration has at least one preset with valid IDs and auth material.
func Validate(cfg model.Config) error {
	if len(cfg.Presets) == 0 {
		return errors.New("config must define at least one preset")
	}
	seen := map[string]struct{}{}
	for _, p := range cfg.Presets {
		if p.ID == "" {
			return errors.New("preset id cannot be empty")
		}
		if _, ok := seen[p.ID]; ok {
			return fmt.Errorf("duplicate preset id: %s", p.ID)
		}
		seen[p.ID] = struct{}{}
		if noAuthMaterial(p) {
			return fmt.Errorf("preset %s must define authKey/authKeyEnv or ephemeralAuthKey/ephemeralAuthKeyEnv", p.ID)
		}
	}
	if cfg.DefaultPreset != "" {
		if _, ok := seen[cfg.DefaultPreset]; !ok {
			return fmt.Errorf("defaultPreset %q not found in presets", cfg.DefaultPreset)
		}
	}
	return nil
}

// FindPreset returns the preset matching the given id, the default preset, or the first preset.
func FindPreset(cfg model.Config, id string) (model.Preset, error) {
	if id == "" {
		id = cfg.DefaultPreset
	}
	if id == "" && len(cfg.Presets) > 0 {
		return cfg.Presets[0], nil
	}
	for _, p := range cfg.Presets {
		if p.ID == id {
			return p, nil
		}
	}
	return model.Preset{}, fmt.Errorf("preset %q not found", id)
}

// ResolvePath returns an absolute path by joining baseDir with the given relative path.
func ResolvePath(baseDir, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

// ResolvePresetSecrets fills in secret values from environment variables for each preset field.
func ResolvePresetSecrets(p model.Preset) model.Preset {
	out := p
	if strings.TrimSpace(out.AuthKey) == "" && strings.TrimSpace(out.AuthKeyEnv) != "" {
		out.AuthKey = strings.TrimSpace(os.Getenv(out.AuthKeyEnv))
	}
	if strings.TrimSpace(out.EphemeralAuthKey) == "" && strings.TrimSpace(out.EphemeralAuthKeyEnv) != "" {
		out.EphemeralAuthKey = strings.TrimSpace(os.Getenv(out.EphemeralAuthKeyEnv))
	}
	if strings.TrimSpace(out.Cleanup.APIKey) == "" && strings.TrimSpace(out.Cleanup.APIKeyEnv) != "" {
		out.Cleanup.APIKey = strings.TrimSpace(os.Getenv(out.Cleanup.APIKeyEnv))
	}
	return out
}

func noAuthMaterial(p model.Preset) bool {
	authInline := strings.TrimSpace(p.AuthKey) != ""
	authEnv := strings.TrimSpace(p.AuthKeyEnv) != ""
	ephInline := strings.TrimSpace(p.EphemeralAuthKey) != ""
	ephEnv := strings.TrimSpace(p.EphemeralAuthKeyEnv) != ""
	return !(authInline || authEnv || ephInline || ephEnv)
}
