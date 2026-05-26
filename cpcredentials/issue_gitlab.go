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

// IssueGitLabParams is the JSON body for POST /agents/credentials/:organization_uuid/:token when requesting GitLab material.
type IssueGitLabParams struct {
	HTTPOrigin       string
	OrganizationUUID string
	ConnectionToken  string
	SkillID          string
	RunID            string
	StepID           string
	UseCaseID        string
	Environment      string
}

// IssuedGitLab holds a short-lived GitLab token returned by the control plane (possibly via delegated AWS execution).
type IssuedGitLab struct {
	Token     string
	ExpiresAt time.Time
}

// IssueGitLab requests a GitLab token for one skill invocation (same HTTP path as AWS issuance; body includes target=gitlab).
func IssueGitLab(ctx context.Context, p IssueGitLabParams) (*IssuedGitLab, error) {
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
		"target":      "gitlab",
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
		GitLab *struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"gitlab"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("decode gitlab credentials response: %w", err)
	}
	if envelope.GitLab == nil {
		return nil, fmt.Errorf("gitlab credentials response missing gitlab object")
	}
	gl := envelope.GitLab
	t := strings.TrimSpace(gl.Token)
	if t == "" {
		return nil, fmt.Errorf("gitlab credentials response missing token")
	}
	out := &IssuedGitLab{Token: t}
	if s := strings.TrimSpace(gl.ExpiresAt); s != "" {
		if ts, err := time.Parse(time.RFC3339, s); err == nil {
			out.ExpiresAt = ts
		} else if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
			out.ExpiresAt = ts
		}
	}
	return out, nil
}
