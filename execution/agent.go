package execution

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fluid/agents/core/skillresult"
	"fluid/agents/core/ws"
)

type SkillExecutor func(skill string, payload map[string]interface{}, meta map[string]interface{}) (map[string]interface{}, error)

type Config struct {
	WebSocketURL     string
	OrganizationUUID string
	Token            string
	Name             string
	AllowedSkills    int
	LogEventsEnabled bool
	LogVerbosity     string
	LogQueueSize     int
	// RuntimeConfig is persisted on the control plane (mirrors agent.yml skills / agent metadata).
	RuntimeConfig map[string]interface{}
	// AfterConnect runs once after the WebSocket handshake succeeds and before skill_invoke handling.
	AfterConnect func() error
}

type Agent struct {
	cfg      Config
	wsClient *ws.Client
	wsMu     sync.Mutex
	stop     chan struct{}
	exec     SkillExecutor
	logQueue chan map[string]interface{}
	logSeq   uint64
}

func New(cfg Config, exec SkillExecutor) *Agent {
	if cfg.LogQueueSize <= 0 {
		cfg.LogQueueSize = 512
	}
	if strings.TrimSpace(cfg.LogVerbosity) == "" {
		cfg.LogVerbosity = "normal"
	}
	return &Agent{
		cfg:      cfg,
		stop:     make(chan struct{}),
		exec:     exec,
		logQueue: make(chan map[string]interface{}, cfg.LogQueueSize),
	}
}

func (a *Agent) Start() error {
	c, err := a.connect()
	if err != nil {
		return fmt.Errorf("control plane connection required but failed: %w", err)
	}
	if a.cfg.AfterConnect != nil {
		if err := a.cfg.AfterConnect(); err != nil {
			_ = c.Close()
			a.closeActiveConnection()
			return fmt.Errorf("after connect: %w", err)
		}
	}
	if a.cfg.LogEventsEnabled {
		go a.logPublishLoop()
	}
	log.Printf("%s execution agent started (skills=%d)", a.cfg.Name, a.cfg.AllowedSkills)
	go a.connectionLoop(c)
	return nil
}

const maxSkillInvokePayloadLog = 8192

func logSkillInvokePayload(skill, ref string, payload map[string]interface{}) {
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Printf("skill_invoke received: skill=%s ref=%s payload=<marshal error: %v>", skill, ref, err)
		return
	}
	s := string(raw)
	if len(s) > maxSkillInvokePayloadLog {
		log.Printf("skill_invoke received: skill=%s ref=%s payload_json=%s... [%d bytes total]",
			skill, ref, s[:maxSkillInvokePayloadLog], len(s))
		return
	}
	log.Printf("skill_invoke received: skill=%s ref=%s payload_json=%s", skill, ref, s)
}

func (a *Agent) Stop() {
	log.Printf("stopping %s execution agent", a.cfg.Name)
	close(a.stop)
	a.closeActiveConnection()
}

func (a *Agent) connectionLoop(c *ws.Client) {
	backoff := 1 * time.Second
	const maxBackoff = 60 * time.Second

	for {
		if c == nil {
			if !sleepOrStop(a.stop, backoff) {
				return
			}
			log.Printf("attempting to reconnect to Fluid control plane (backoff: %v)", backoff)
			next, err := a.connect()
			if err != nil {
				log.Printf("reconnection failed: %v", err)
				backoff = minDuration(backoff*2, maxBackoff)
				continue
			}
			log.Printf("successfully reconnected to Fluid control plane")
			if a.cfg.AfterConnect != nil {
				if err := a.cfg.AfterConnect(); err != nil {
					log.Printf("reattaching to control plane: after connect failed: %v", err)
					_ = next.Close()
					a.closeActiveConnection()
					backoff = minDuration(backoff*2, maxBackoff)
					continue
				}
				log.Printf("reattaching to control plane: runtime sync complete")
			}
			c = next
		}

		if !a.runConnectionSession(c) {
			return
		}
		c = nil
		backoff = 1 * time.Second
	}
}

func (a *Agent) runConnectionSession(c *ws.Client) bool {
	errCh := make(chan error, 2)
	done := make(chan struct{})

	go a.pingLoop(c, done, errCh)
	go a.readLoop(c, done, errCh)

	select {
	case <-a.stop:
		close(done)
		_ = c.Close()
		return false
	case err := <-errCh:
		close(done)
		log.Printf("connection lost: %v", err)
		_ = c.Close()
		return true
	}
}

func (a *Agent) pingLoop(c *ws.Client, done <-chan struct{}, errCh chan<- error) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-done:
			return
		case <-t.C:
			if err := c.SendPing(); err != nil {
				nonBlockingErrSend(errCh, err)
				return
			}
		}
	}
}

func (a *Agent) readLoop(c *ws.Client, done <-chan struct{}, errCh chan<- error) {
	for {
		select {
		case <-done:
			return
		default:
		}

		msg, err := c.Read()
		if err != nil {
			nonBlockingErrSend(errCh, err)
			return
		}
		if msg.Event != "skill_invoke" {
			continue
		}

		ref, _ := msg.Payload["ref"].(string)
		skill, _ := msg.Payload["skill"].(string)
		payload, _ := msg.Payload["payload"].(map[string]interface{})
		if payload == nil {
			payload = map[string]interface{}{}
		}
		logSkillInvokePayload(skill, ref, payload)
		meta, _ := msg.Payload["meta"].(map[string]interface{})
		a.enqueueLogEvent(meta, "info", "system", fmt.Sprintf("skill_invoke received skill=%s ref=%s", skill, ref))
		stopFileForwarder := a.startFileLogForwarder(meta, payload)

		if meta == nil {
			meta = map[string]interface{}{}
		}
		result, runErr := a.exec(skill, payload, meta)
		stopFileForwarder()
		if runErr != nil {
			log.Printf("skill failed: skill=%s ref=%s err=%v", skill, ref, runErr)
			a.enqueueLogEvent(meta, "error", "system", fmt.Sprintf("skill failed skill=%s ref=%s err=%v", skill, ref, runErr))
			result = skillresult.Failure(runErr.Error())
		}
		a.enqueueSkillOutputTails(meta, result)

		if err := c.SendSkillResult(ref, result); err != nil {
			a.enqueueLogEvent(meta, "error", "system", fmt.Sprintf("skill_result send failed skill=%s ref=%s err=%v", skill, ref, err))
			nonBlockingErrSend(errCh, err)
			return
		}
		log.Printf("skill_result sent: skill=%s ref=%s", skill, ref)
		a.enqueueLogEvent(meta, "info", "system", fmt.Sprintf("skill_result sent skill=%s ref=%s", skill, ref))
	}
}

func (a *Agent) startFileLogForwarder(meta map[string]interface{}, payload map[string]interface{}) func() {
	if !a.cfg.LogEventsEnabled || payload == nil {
		return func() {}
	}
	runID, _ := meta["run_id"].(string)
	if strings.TrimSpace(runID) == "" {
		return func() {}
	}
	logPath, _ := payload["fluid_log_path"].(string)
	safePath, err := safeFluidLogPath(logPath)
	if err != nil {
		log.Printf("fluid_log_path rejected: %v", err)
		return func() {}
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		var lastSize int64 = 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			info, err := os.Stat(safePath)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			size := info.Size()
			if size < lastSize {
				lastSize = 0
			}
			if size > lastSize {
				b, err := os.ReadFile(safePath)
				if err == nil {
					chunk := b[int(lastSize):]
					for _, line := range splitLogLines(string(chunk)) {
						if strings.TrimSpace(line) != "" {
							a.enqueueLogEvent(meta, "info", "script_log", line)
						}
					}
				}
				lastSize = size
			}
			time.Sleep(1 * time.Second)
		}
	}()

	return func() {
		close(stop)
		<-done
	}
}

func splitLogLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (a *Agent) enqueueSkillOutputTails(meta map[string]interface{}, result map[string]interface{}) {
	if !a.cfg.LogEventsEnabled || result == nil {
		return
	}
	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return
	}
	if s, ok := data["stdout_tail"].(string); ok && strings.TrimSpace(s) != "" {
		a.enqueueLogEvent(meta, "info", "stdout", s)
	}
	if s, ok := data["stderr_tail"].(string); ok && strings.TrimSpace(s) != "" {
		a.enqueueLogEvent(meta, "error", "stderr", s)
	}
}

func (a *Agent) enqueueLogEvent(meta map[string]interface{}, level, stream, message string) {
	if !a.cfg.LogEventsEnabled || strings.TrimSpace(message) == "" {
		return
	}
	seq := atomic.AddUint64(&a.logSeq, 1)
	event := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339Nano),
		"event_ts":       time.Now().UTC().Format(time.RFC3339Nano),
		"seq":            seq,
		"level":          level,
		"stream":         stream,
		"verbosity":      a.cfg.LogVerbosity,
		"message":        message,
		"execution_role": a.cfg.Name,
	}
	if meta != nil {
		if v, ok := meta["run_id"]; ok {
			event["use_case_run_id"] = v
		}
		if v, ok := meta["step_id"]; ok {
			event["step_id"] = v
		}
		if v, ok := meta["skill_id"]; ok {
			event["skill_id"] = v
		}
	}
	select {
	case a.logQueue <- event:
	default:
		// Drop when queue is full to avoid blocking skill execution.
	}
}

func (a *Agent) logPublishLoop() {
	backoff := time.Second
	for {
		select {
		case <-a.stop:
			return
		case event := <-a.logQueue:
			if err := a.sendLogEvent(event); err != nil {
				time.Sleep(backoff)
				if backoff < 5*time.Second {
					backoff *= 2
				}
				select {
				case a.logQueue <- event:
				default:
				}
				continue
			}
			backoff = time.Second
		}
	}
}

func (a *Agent) sendLogEvent(event map[string]interface{}) error {
	a.wsMu.Lock()
	client := a.wsClient
	a.wsMu.Unlock()
	if client == nil {
		return fmt.Errorf("no websocket client")
	}
	return client.SendLogEvent(event)
}

func (a *Agent) connect() (*ws.Client, error) {
	log.Printf("connecting to Fluid websocket at %s", a.cfg.WebSocketURL)
	c, err := ws.Connect(a.cfg.WebSocketURL, a.cfg.OrganizationUUID, a.cfg.Token)
	if err != nil {
		return nil, err
	}
	a.setActiveConnection(c)

	log.Printf("connected to Fluid control plane (org=%s)", a.cfg.OrganizationUUID)

	if len(a.cfg.RuntimeConfig) > 0 {
		if err := c.SendPushRuntimeConfig(a.cfg.RuntimeConfig); err != nil {
			log.Printf("push runtime_config to control plane failed: %v", err)
		} else {
			log.Printf("runtime_config synced from local agent definition (skills=%d)", a.cfg.AllowedSkills)
		}
	}
	return c, nil
}

func (a *Agent) setActiveConnection(c *ws.Client) {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	a.wsClient = c
}

func (a *Agent) closeActiveConnection() {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	if a.wsClient != nil {
		_ = a.wsClient.Close()
		a.wsClient = nil
	}
}

func nonBlockingErrSend(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}

func sleepOrStop(stop <-chan struct{}, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-stop:
		return false
	case <-t.C:
		return true
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
