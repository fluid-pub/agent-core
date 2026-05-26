package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const joinWait = 15 * time.Second

type Message struct {
	Topic   string                 `json:"topic"`
	Event   string                 `json:"event"`
	Payload map[string]interface{} `json:"payload"`
	Ref     string                 `json:"ref"`
}

type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
	ref  int64
}

func Connect(baseWSURL, orgUUID, token string) (*Client, error) {
	u, err := url.Parse(baseWSURL)
	if err != nil {
		return nil, fmt.Errorf("parse websocket url: %w", err)
	}

	q := u.Query()
	q.Set("organization_uuid", orgUUID)
	q.Set("token", token)
	u.RawQuery = q.Encode()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return nil, fmt.Errorf("dial websocket: %w", err)
	}

	c := &Client{conn: conn}
	joinRef := c.nextRef()
	if err := c.send(Message{
		Topic:   "agent:lobby",
		Event:   "phx_join",
		Payload: map[string]interface{}{},
		Ref:     joinRef,
	}); err != nil {
		return nil, err
	}

	if err := c.waitJoinOK(joinRef); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) waitJoinOK(joinRef string) error {
	deadline := time.Now().Add(joinWait)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("phx_join: timeout waiting for reply")
		}
		if err := c.conn.SetReadDeadline(time.Now().Add(remaining)); err != nil {
			return fmt.Errorf("phx_join: %w", err)
		}

		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("phx_join: read: %w", err)
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("phx_join: decode: %w", err)
		}

		if msg.Event != "phx_reply" || msg.Ref != joinRef {
			continue
		}

		status, _ := msg.Payload["status"].(string)
		if status == "ok" {
			_ = c.conn.SetReadDeadline(time.Time{})
			return nil
		}

		reason := msg.Payload["response"]
		return fmt.Errorf("phx_join failed: status=%s response=%v", status, reason)
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendPing() error {
	return c.send(Message{
		Topic:   "agent:lobby",
		Event:   "ping",
		Payload: map[string]interface{}{},
		Ref:     c.nextRef(),
	})
}

func (c *Client) SendSkillResult(ref string, result map[string]interface{}) error {
	return c.send(Message{
		Topic: "agent:lobby",
		Event: "skill_result",
		Payload: map[string]interface{}{
			"ref":    ref,
			"result": result,
		},
		Ref: c.nextRef(),
	})
}

// SendPushRuntimeConfig sends the execution agent snapshot (e.g. skills from agent.yml) to the control plane.
func (c *Client) SendPushRuntimeConfig(runtimeConfig map[string]interface{}) error {
	if len(runtimeConfig) == 0 {
		return nil
	}
	return c.send(Message{
		Topic: "agent:lobby",
		Event: "push_runtime_config",
		Payload: map[string]interface{}{
			"runtime_config": runtimeConfig,
		},
		Ref: c.nextRef(),
	})
}

func (c *Client) SendLogEvent(event map[string]interface{}) error {
	return c.send(Message{
		Topic: "agent:lobby",
		Event: "push_log_event",
		Payload: map[string]interface{}{
			"event": event,
		},
		Ref: c.nextRef(),
	})
}

func (c *Client) Read() (Message, error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return Message{}, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

func (c *Client) send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, b)
}

func (c *Client) nextRef() string {
	c.ref++
	return fmt.Sprintf("%d", c.ref)
}
