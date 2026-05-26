package enroll

import (
	"os"
	"strings"
)

// RunIDFromPrefetchSources returns the use case run id used for control-plane credential prefetch
// (FLUID_USE_CASE_RUN_ID or use_case_run_id inside FLUID_ENROLLMENT_EXTRA_ARGS JSON).
func RunIDFromPrefetchSources() string {
	if s := strings.TrimSpace(os.Getenv("FLUID_USE_CASE_RUN_ID")); s != "" {
		return s
	}
	extra, err := ExtraArgsFromEnv()
	if err != nil || extra == nil {
		return ""
	}
	return strings.TrimSpace(extra["use_case_run_id"])
}
