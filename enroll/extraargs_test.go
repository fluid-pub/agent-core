package enroll

import (
	"os"
	"testing"
)

func TestExtraArgsFromEnvUnset(t *testing.T) {
	_ = os.Unsetenv(envEnrollmentExtraArgs)
	m, err := ExtraArgsFromEnv()
	if err != nil || m != nil {
		t.Fatalf("got %v %v", m, err)
	}
}

func TestExtraArgsFromEnvJSON(t *testing.T) {
	t.Setenv(envEnrollmentExtraArgs, `{"use_case_run_id":"abc","execution_role":"aws","n":42}`)
	m, err := ExtraArgsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if m["use_case_run_id"] != "abc" || m["execution_role"] != "aws" || m["n"] != "42" {
		t.Fatalf("%#v", m)
	}
}
