package cpcredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IssueAWSParams is the JSON body for POST /agents/credentials/:organization_uuid/:token plus path segments.
type IssueAWSParams struct {
	HTTPOrigin       string
	OrganizationUUID string
	ConnectionToken  string
	SkillID          string
	RunID            string
	StepID           string
	UseCaseID        string
	Environment      string // empty means "default"
}

// IssuedAWS holds STS keys returned by the control plane (AssumeRoleWithWebIdentity on the CP side).
type IssuedAWS struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	// ExpiresAt is set when the control plane includes expires_at (RFC3339) on the issuance envelope.
	ExpiresAt time.Time
}

// IssueAWS requests short-lived AWS credentials for one skill invocation (workload identity on the CP).
func IssueAWS(ctx context.Context, p IssueAWSParams) (*IssuedAWS, error) {
	base := strings.TrimRight(strings.TrimSpace(p.HTTPOrigin), "/")
	if base == "" {
		return nil, fmt.Errorf("http origin is required")
	}
	org := strings.TrimSpace(p.OrganizationUUID)
	tok := strings.TrimSpace(p.ConnectionToken)
	if org == "" || tok == "" {
		return nil, fmt.Errorf("organization_uuid and connection token are required")
	}
	skill := strings.TrimSpace(p.SkillID)
	if skill == "" {
		return nil, fmt.Errorf("skill_id is required")
	}
	run := strings.TrimSpace(p.RunID)
	if run == "" {
		return nil, fmt.Errorf("run_id is required for workload credentials")
	}

	u, err := url.Parse(base + "/agents/credentials/" + url.PathEscape(org) + "/" + url.PathEscape(tok))
	if err != nil {
		return nil, fmt.Errorf("build credentials url: %w", err)
	}

	env := strings.TrimSpace(p.Environment)
	if env == "" {
		env = "default"
	}
	body := map[string]string{
		"target":      "aws",
		"skill_id":    skill,
		"run_id":      run,
		"step_id":     strings.TrimSpace(p.StepID),
		"use_case_id": strings.TrimSpace(p.UseCaseID),
		"environment": env,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal credentials request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST credentials: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512<<10))

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("credentials issue: HTTP %d: invalid agent token", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("credentials issue: HTTP %d: no credential binding for skill %q", resp.StatusCode, skill)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("credentials issue: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var envelope struct {
		ExpiresAtRaw *string `json:"expires_at"`
		Credentials  *struct {
			AccessKeyID     string `json:"access_key_id"`
			SecretAccessKey string `json:"secret_access_key"`
			SessionToken    string `json:"session_token"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("decode credentials response: %w", err)
	}
	if envelope.Credentials == nil {
		return nil, fmt.Errorf("credentials response missing credentials object")
	}
	c := envelope.Credentials
	if strings.TrimSpace(c.AccessKeyID) == "" || strings.TrimSpace(c.SecretAccessKey) == "" {
		return nil, fmt.Errorf("credentials response missing access_key_id or secret_access_key")
	}
	out := &IssuedAWS{
		AccessKeyID:     strings.TrimSpace(c.AccessKeyID),
		SecretAccessKey: strings.TrimSpace(c.SecretAccessKey),
		SessionToken:    strings.TrimSpace(c.SessionToken),
	}
	if envelope.ExpiresAtRaw != nil {
		raw := strings.TrimSpace(*envelope.ExpiresAtRaw)
		if raw != "" {
			if ts, err := time.Parse(time.RFC3339, raw); err == nil {
				out.ExpiresAt = ts
			} else if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
				out.ExpiresAt = ts
			}
		}
	}
	return out, nil
}
