package cpcredentials

import (
	"fmt"
	"net/url"
	"strings"
)

// HTTPOriginFromWebSocketURL returns the control plane HTTP origin (scheme + host) for REST calls
// such as POST /agents/credentials/... (no path from the WebSocket URL is preserved).
func HTTPOriginFromWebSocketURL(wsURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(wsURL))
	if err != nil {
		return "", fmt.Errorf("parse websocket url: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("websocket url has no host")
	}
	scheme := "https"
	if u.Scheme == "ws" {
		scheme = "http"
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return "", fmt.Errorf("websocket url scheme must be ws or wss, got %q", u.Scheme)
	}
	return scheme + "://" + u.Host, nil
}
