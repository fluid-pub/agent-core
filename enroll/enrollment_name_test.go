package enroll

import (
	"strings"
	"testing"
)

func TestResolveExecutionAgentEnrollmentName(t *testing.T) {
	if got := ResolveExecutionAgentEnrollmentName("host", "aws", "  my-aws  "); got != "my-aws" {
		t.Fatalf("override got %q", got)
	}
	if got := ResolveExecutionAgentEnrollmentName("host", "aws", ""); got != "host-aws" {
		t.Fatalf("default got %q", got)
	}
	long := strings.Repeat("b", 120)
	if got := ResolveExecutionAgentEnrollmentName("x", "aws", long); len(got) != 100 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestDefaultChildExecutionAgentName(t *testing.T) {
	if got := DefaultChildExecutionAgentName("dependabot-mr-694-1039", "aws"); got != "dependabot-mr-694-1039-aws" {
		t.Fatalf("got %q", got)
	}
	if got := DefaultChildExecutionAgentName("  host  ", "gitlab"); got != "host-gitlab" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("a", 98)
	got := DefaultChildExecutionAgentName(long, "aws")
	if len(got) != 100 || !strings.HasSuffix(got, "-aws") {
		t.Fatalf("len=%d got %q", len(got), got)
	}
}
