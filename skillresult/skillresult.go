// Package skillresult defines the JSON envelope for skill_result payloads sent to the Fluid control plane.
//
// Contract (all keys are JSON strings):
//
//	Success: {"ok": true, "data": { ... skill-specific fields ... }}
//	Failure: {"ok": false, "error": "message"}
//
// Execution errors from SkillExecutor (non-nil error) are wrapped the same way in readLoop.
package skillresult

const (
	KeyOK    = "ok"
	KeyError = "error"
	KeyData  = "data"
)

// Success returns the standard success envelope. data holds skill-specific fields (e.g. zones, mr_url).
func Success(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	return map[string]interface{}{
		KeyOK:   true,
		KeyData: data,
	}
}

// Failure returns a logical failure without a Go error (rare); prefer returning (nil, err) from the executor.
func Failure(message string) map[string]interface{} {
	return map[string]interface{}{
		KeyOK:    false,
		KeyError: message,
	}
}

// FailureWithData returns a failure envelope that also carries structured details
// (for example stdout/stderr tails from linux.script.run).
func FailureWithData(message string, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	return map[string]interface{}{
		KeyOK:    false,
		KeyError: message,
		KeyData:  data,
	}
}
