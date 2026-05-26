package enroll

import "testing"

func TestRunIDFromPrefetchSourcesEnv(t *testing.T) {
	t.Setenv("FLUID_USE_CASE_RUN_ID", "run-from-env")
	t.Setenv(envEnrollmentExtraArgs, "")
	if got := RunIDFromPrefetchSources(); got != "run-from-env" {
		t.Fatalf("got %q", got)
	}
}

func TestRunIDFromPrefetchSourcesExtraArgs(t *testing.T) {
	t.Setenv("FLUID_USE_CASE_RUN_ID", "")
	t.Setenv(envEnrollmentExtraArgs, `{"use_case_run_id":"run-from-json"}`)
	if got := RunIDFromPrefetchSources(); got != "run-from-json" {
		t.Fatalf("got %q", got)
	}
}
