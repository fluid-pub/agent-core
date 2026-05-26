package enroll

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const envEnrollmentExtraArgs = "FLUID_ENROLLMENT_EXTRA_ARGS"

// ExtraArgsFromEnv parses FLUID_ENROLLMENT_EXTRA_ARGS as a JSON object (e.g. {"use_case_run_id":"...","execution_role":"gitlab"}).
// Keys and values are normalized to strings for the enrollment API extra_args map.
// Returns nil, nil when the variable is unset or blank.
func ExtraArgsFromEnv() (map[string]string, error) {
	raw := strings.TrimSpace(os.Getenv(envEnrollmentExtraArgs))
	if raw == "" {
		return nil, nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("%s: %w", envEnrollmentExtraArgs, err)
	}
	out := make(map[string]string, len(decoded))
	for k, v := range decoded {
		if strings.TrimSpace(k) == "" {
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = t
		case float64:
			out[k] = fmt.Sprintf("%.0f", t)
		case bool:
			out[k] = fmt.Sprintf("%t", t)
		case nil:
			out[k] = ""
		default:
			out[k] = fmt.Sprint(t)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
