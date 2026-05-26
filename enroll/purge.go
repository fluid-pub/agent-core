package enroll

import (
	"os"
	"strings"
)

const envPurgeEnrollmentSources = "FLUID_ENROLL_PURGE_ENROLLMENT_SOURCES"

// PurgeEnrollmentSources reports whether enrollment source files (e.g. systemd EnvironmentFile)
// should be removed after a successful exchange. Default is true (secure for long-lived hosts).
// Set FLUID_ENROLL_PURGE_ENROLLMENT_SOURCES to false, 0, no, or off to keep the enrollment secret
// on disk so another process on the same machine can reuse the same enrollment token (until max_uses).
func PurgeEnrollmentSources() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(envPurgeEnrollmentSources)))
	if v == "" {
		return true
	}
	switch v {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}
