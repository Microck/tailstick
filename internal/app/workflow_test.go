package app

import (
	"testing"
	"time"

	"github.com/tailstick/tailstick/internal/model"
)

func TestBuildDeviceName(t *testing.T) {
	got := buildDeviceName(model.LeaseModeTimed, "ops-read", "finance-laptop", "abc123", "night")
	want := "tsusb-timed-ops-read-finance-laptop-abc123-night"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShouldCleanupSessionOnBootChange(t *testing.T) {
	rec := model.LeaseRecord{
		Mode:          model.LeaseModeSession,
		CreatedBootID: "boot-a",
		Status:        model.LeaseStatusActive,
	}
	if !shouldCleanup(rec, "boot-b", time.Now().UTC()) {
		t.Fatalf("expected session lease to cleanup after boot id change")
	}
}

func TestShouldCleanupTimedOnExpiry(t *testing.T) {
	exp := time.Now().UTC().Add(-1 * time.Minute)
	rec := model.LeaseRecord{
		Mode:      model.LeaseModeTimed,
		ExpiresAt: &exp,
		Status:    model.LeaseStatusActive,
	}
	if !shouldCleanup(rec, "same-boot", time.Now().UTC()) {
		t.Fatalf("expected timed lease to cleanup on expiry")
	}
}
