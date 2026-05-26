package execution

import (
	"fmt"
	"path/filepath"
	"strings"
)

// fluidLogPathRoot is where the control plane places per-run logs (see UseCaseEngine.FluidVars).
const fluidLogPathRoot = "/tmp/fluid"

// safeFluidLogPath validates fluid_log_path from skill payloads before os.Stat/os.ReadFile.
// Only absolute paths under /tmp/fluid/ are allowed (no traversal via ..).
func safeFluidLogPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty log path")
	}
	if strings.Contains(raw, "\x00") {
		return "", fmt.Errorf("invalid log path")
	}
	if !filepath.IsAbs(raw) {
		return "", fmt.Errorf("log path must be absolute")
	}

	cleaned := filepath.Clean(raw)
	root := filepath.Clean(fluidLogPathRoot)
	if cleaned != root && !strings.HasPrefix(cleaned, root+string(filepath.Separator)) {
		return "", fmt.Errorf("log path must be under %s", fluidLogPathRoot)
	}

	rel, err := filepath.Rel(root, cleaned)
	if err != nil {
		return "", fmt.Errorf("log path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("log path escapes allowed directory")
	}

	return cleaned, nil
}
