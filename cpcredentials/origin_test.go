package cpcredentials

import "testing"

func TestHTTPOriginFromWebSocketURL(t *testing.T) {
	got, err := HTTPOriginFromWebSocketURL("wss://dev.fluid.pub/v1/agents/websocket?organization_uuid=x&token=y")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://dev.fluid.pub" {
		t.Fatalf("got %q", got)
	}
	got, err = HTTPOriginFromWebSocketURL("ws://localhost:4000/v1/agents/websocket")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://localhost:4000" {
		t.Fatalf("got %q", got)
	}
}
