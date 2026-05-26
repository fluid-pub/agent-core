package enroll

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const enrollPath = "/api/v1/enrollment/enroll"

// Principal is the resource family created when redeeming a token (control plane contract).
type Principal string

const (
	PrincipalExecutionAgent Principal = "execution_agent"
	PrincipalProbe          Principal = "probe"
)

// Result is the successful JSON body from POST /api/v1/enrollment/enroll.
type Result struct {
	ResourceKind      string `json:"resource_kind"`
	ResourceID        string `json:"resource_id"`
	ConnectionToken   string `json:"connection_token"`
	OrganizationUUID  string `json:"organization_uuid"`
	OrganizationID    string `json:"organization_id"`
	EnrollmentTokenID string `json:"enrollment_token_id"`
	Status            string `json:"status"`
	UseCount          int    `json:"use_count"`
}

// Params for the enrollment exchange.
type Params struct {
	// BaseURL is the control plane origin, e.g. https://cp.example (no trailing slash required).
	BaseURL string
	// EnrollmentToken is the plaintext token from provisioning.
	EnrollmentToken string
	// Hostname is sent as "hostname" when Name is empty so the server can derive the display name.
	Hostname string
	// Name is an optional explicit display name.
	Name string
	// Principal is execution_agent or probe (defaults to execution_agent if empty).
	Principal Principal
	// AgentType is the catalog slug, e.g. linux, gitlab, aws, proxmox.
	AgentType string
	// ExtraArgs is sent as the JSON "extra_args" object when non-empty (must match token extra_args_bind when set).
	ExtraArgs map[string]string
}

// Exchange calls POST /api/v1/enrollment/enroll and returns credentials for WebSocket registration.
func Exchange(ctx context.Context, p Params) (*Result, error) {
	base := strings.TrimSpace(p.BaseURL)
	base = strings.TrimRight(base, "/")
	token := strings.TrimSpace(p.EnrollmentToken)
	if base == "" || token == "" {
		return nil, fmt.Errorf("enroll: base URL and enrollment token are required")
	}
	agentType := strings.TrimSpace(p.AgentType)
	if agentType == "" {
		return nil, fmt.Errorf("enroll: agent_type is required")
	}
	principal := p.Principal
	if principal == "" {
		principal = PrincipalExecutionAgent
	}

	body := map[string]interface{}{
		"enrollment_token": token,
		"principal":        string(principal),
		"agent_type":       agentType,
	}
	if hn := strings.TrimSpace(p.Hostname); hn != "" {
		body["hostname"] = hn
	}
	if n := strings.TrimSpace(p.Name); n != "" {
		body["name"] = n
	}
	if len(p.ExtraArgs) > 0 {
		body["extra_args"] = p.ExtraArgs
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("enroll: marshal body: %w", err)
	}

	url := base + enrollPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("enroll: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("enroll: POST %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("enroll: read body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf(
			"enroll: POST %s returned HTTP %d: %s",
			url,
			resp.StatusCode,
			strings.TrimSpace(string(respBody)),
		)
	}

	var out Result
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("enroll: decode response: %w", err)
	}
	if strings.TrimSpace(out.ConnectionToken) == "" {
		return nil, fmt.Errorf("enroll: missing connection_token in response")
	}
	orgUUID := strings.TrimSpace(out.OrganizationUUID)
	if orgUUID == "" {
		orgUUID = strings.TrimSpace(out.OrganizationID)
	}
	if orgUUID == "" {
		return nil, fmt.Errorf("enroll: missing organization_uuid in response")
	}
	out.OrganizationUUID = orgUUID
	return &out, nil
}
