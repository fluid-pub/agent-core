package execution

import (
	"path/filepath"
	"testing"
)

func TestSafeFluidLogPath(t *testing.T) {
	runID := "550e8400-e29b-41d4-a716-446655440000"
	valid := filepath.Join(fluidLogPathRoot, runID, "logs", "use-case.log")

	got, err := safeFluidLogPath(valid)
	if err != nil {
		t.Fatalf("expected valid path: %v", err)
	}
	if got != filepath.Clean(valid) {
		t.Fatalf("got %q want %q", got, filepath.Clean(valid))
	}

	cases := []string{
		"",
		"relative/log",
		"/etc/passwd",
		"/tmp/fluid/../etc/passwd",
		"/tmp/fluid/../../etc/passwd",
	}
	for _, p := range cases {
		if _, err := safeFluidLogPath(p); err == nil {
			t.Fatalf("expected error for %q", p)
		}
	}
}
