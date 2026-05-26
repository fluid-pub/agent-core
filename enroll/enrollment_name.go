package enroll

import "strings"

const maxAgentNameLen = 100

// EnvEnrollmentName is the preferred env var for the execution agent name sent as JSON "name" on
// POST /api/v1/enrollment/enroll (same as control plane Agent.name, unique per organization).
// Set by the child from YAML enrollment.name (e.g. use case with.agent_config_yaml) or this env
// if still provided by a bootstrap step.
const EnvEnrollmentName = "FLUID_ENROLLMENT_NAME"

// EnvEnrollmentNameDeprecated is the previous env name; child CLIs still honor it when
// FLUID_ENROLLMENT_NAME is unset.
const EnvEnrollmentNameDeprecated = "FLUID_ENROLLMENT_DISPLAY_NAME"

// ResolveExecutionAgentEnrollmentName returns the value for enroll Params.Name.
// Non-empty override (env or YAML enrollment.name) is trimmed and clipped to 100 bytes
// (aligned with clip_agent_display_name on the control plane). Otherwise DefaultChildExecutionAgentName.
func ResolveExecutionAgentEnrollmentName(hostname, agentType, override string) string {
	o := strings.TrimSpace(override)
	if o != "" {
		return clipAgentName(o, maxAgentNameLen)
	}
	return DefaultChildExecutionAgentName(hostname, agentType)
}

// DefaultChildExecutionAgentName is the fallback agent name for a child execution agent on the same
// host as the Linux worker (hostname-only would collide with the Linux agent name).
// Example: hostname "vm-1", agentType "aws" -> "vm-1-aws". Truncates hostname if needed to stay
// within maxAgentNameLen (control plane Agent.name).
func DefaultChildExecutionAgentName(hostname, agentType string) string {
	hn := strings.TrimSpace(hostname)
	at := strings.TrimSpace(strings.ToLower(agentType))
	if at == "" {
		return clipAgentName(hn, maxAgentNameLen)
	}
	suffix := "-" + at
	if hn == "" {
		return clipAgentName("fluid"+suffix, maxAgentNameLen)
	}
	if len(hn)+len(suffix) <= maxAgentNameLen {
		return hn + suffix
	}
	allow := maxAgentNameLen - len(suffix)
	if allow < 1 {
		return clipAgentName(suffix, maxAgentNameLen)
	}
	return hn[:allow] + suffix
}

func clipAgentName(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
