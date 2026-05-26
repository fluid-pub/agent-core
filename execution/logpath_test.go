package execution

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeFluidLogPath_valid(t *testing.T) {
	runID := "550e8400-e29b-41d4-a716-446655440000"
	want := filepath.Join(fluidLogPathRoot, runID, "logs", "use-case.log")

	tests := []struct {
		name string
		in   string
	}{
		{"canonical", want},
		{"trimmed", "  " + want + "  "},
		{"extra_slash", filepath.Join(fluidLogPathRoot, runID, "logs", "", "use-case.log")},
		{"root_child", filepath.Join(fluidLogPathRoot, "child", "log.txt")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeFluidLogPath(tt.in)
			if err != nil {
				t.Fatalf("safeFluidLogPath: %v", err)
			}
			if got != filepath.Clean(strings.TrimSpace(tt.in)) {
				t.Fatalf("got %q want %q", got, filepath.Clean(strings.TrimSpace(tt.in)))
			}
			if !strings.HasPrefix(got, fluidLogPathRoot+string(filepath.Separator)) {
				t.Fatalf("result not under root: %q", got)
			}
		})
	}
}

func TestSafeFluidLogPath_rejects(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"relative", "tmp/fluid/run/log"},
		{"relative_with_dot", "./tmp/fluid/run/log"},
		{"outside_etc", "/etc/passwd"},
		{"outside_var", "/var/log/syslog"},
		{"prefix_sibling", "/tmp/fluid-evil/run/log"},
		{"prefix_sibling_no_slash", "/tmp/fluid2/run/log"},
		{"dotdot_after_root", filepath.Join(fluidLogPathRoot, "..", "etc", "passwd")},
		{"dotdot_mid", filepath.Join(fluidLogPathRoot, "run", "..", "..", "etc", "passwd")},
		{"encoded_traversal", "/tmp/fluid/../etc/passwd"},
		{"double_encoded", "/tmp/fluid/run/../../etc/passwd"},
		{"null_byte", "/tmp/fluid/run\x00/log"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := safeFluidLogPath(tt.path)
			if err == nil {
				t.Fatalf("expected error for path %q", tt.path)
			}
		})
	}
}

func TestSafeFluidLogPath_rejectsPathOutsideRootMessage(t *testing.T) {
	_, err := safeFluidLogPath("/etc/passwd")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), fluidLogPathRoot) {
		t.Fatalf("error should mention allowed root, got: %v", err)
	}
}

func TestStartFileLogForwarder_rejectsUnsafeLogPath(t *testing.T) {
	a := &Agent{
		cfg: Config{
			LogEventsEnabled: true,
			LogVerbosity:     "normal",
		},
	}
	stop := a.startFileLogForwarder(
		map[string]interface{}{"run_id": "550e8400-e29b-41d4-a716-446655440000"},
		map[string]interface{}{"fluid_log_path": "/etc/passwd"},
	)
	stop()
	// No panic and immediate no-op: unsafe path must not start the tail goroutine.
}
