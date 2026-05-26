package enroll

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != enrollPath || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var got map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got["hostname"] != "vm-1" || got["enrollment_token"] != "tok" || got["agent_type"] != "linux" {
			t.Fatalf("unexpected body: %#v", got)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"resource_kind":       "agent",
			"resource_id":         "550e8400-e29b-41d4-a716-446655440000",
			"connection_token":    "conn-secret",
			"organization_id":     "00000000-0000-0000-0000-000000000001",
			"organization_uuid":   "63483198-c965-44bd-b1bc-9552e05caa37",
			"enrollment_token_id": "550e8400-e29b-41d4-a716-446655440001",
			"status":              "consumed",
			"use_count":           1,
		})
	}))
	defer ts.Close()

	res, err := Exchange(context.Background(), Params{
		BaseURL:         ts.URL,
		EnrollmentToken: "tok",
		Hostname:        "vm-1",
		AgentType:       "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ConnectionToken != "conn-secret" || res.OrganizationUUID != "63483198-c965-44bd-b1bc-9552e05caa37" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestExchangeWithExtraArgs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		ea, ok := got["extra_args"].(map[string]interface{})
		if !ok || ea["use_case_run_id"] != "run-1" || ea["execution_role"] != "gitlab" {
			t.Fatalf("unexpected extra_args: %#v", got["extra_args"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"connection_token":  "ct",
			"organization_uuid": "550e8400-e29b-41d4-a716-446655440099",
		})
	}))
	defer ts.Close()

	res, err := Exchange(context.Background(), Params{
		BaseURL:         ts.URL,
		EnrollmentToken: "tok",
		Hostname:        "h",
		AgentType:       "gitlab",
		ExtraArgs:       map[string]string{"use_case_run_id": "run-1", "execution_role": "gitlab"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ConnectionToken != "ct" {
		t.Fatalf("token: %+v", res)
	}
}

func TestExchangeProbePrincipal(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got["principal"] != "probe" || got["agent_type"] != "linux" {
			t.Fatalf("unexpected: %#v", got)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"connection_token":  "p",
			"organization_uuid": "550e8400-e29b-41d4-a716-446655440088",
		})
	}))
	defer ts.Close()

	_, err := Exchange(context.Background(), Params{
		BaseURL:         ts.URL,
		EnrollmentToken: "t",
		Hostname:        "h",
		Principal:       PrincipalProbe,
		AgentType:       "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExchangeRequiresBaseAndToken(t *testing.T) {
	_, err := Exchange(context.Background(), Params{AgentType: "linux"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExchangeRequiresAgentType(t *testing.T) {
	_, err := Exchange(context.Background(), Params{
		BaseURL:         "http://x",
		EnrollmentToken: "t",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
