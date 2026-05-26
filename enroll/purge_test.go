package enroll

import (
	"os"
	"testing"
)

func TestPurgeEnrollmentSourcesDefault(t *testing.T) {
	_ = os.Unsetenv(envPurgeEnrollmentSources)
	if !PurgeEnrollmentSources() {
		t.Fatal("expected default true")
	}
}

func TestPurgeEnrollmentSourcesFalse(t *testing.T) {
	t.Setenv(envPurgeEnrollmentSources, "false")
	if PurgeEnrollmentSources() {
		t.Fatal("expected false")
	}
	t.Setenv(envPurgeEnrollmentSources, "0")
	if PurgeEnrollmentSources() {
		t.Fatal("expected false")
	}
}
