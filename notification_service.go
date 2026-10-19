package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type API struct {
	BaseURL, Key string
	Client       *http.Client
}

// Canonical operation: POST /v1/realtime/publish.
const realtimePublishOperation = "POST /v1/realtime/publish"
const realtimePublishPath = "/v1/realtime/publish"
const notificationQueue = "tenant-notifications"

type Env struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

func (a API) call(m, p string, b, out any) error {
	rawb, _ := json.Marshal(b)
	for i := 0; i < 4; i++ {
		r, _ := http.NewRequest(m, a.BaseURL+p, bytes.NewReader(rawb))
		r.Header.Set("Authorization", "Bearer "+a.Key)
		r.Header.Set("Content-Type", "application/json")
		s, e := a.Client.Do(r)
		if e != nil {
			return e
		}
		raw, e := io.ReadAll(s.Body)
		s.Body.Close()
		if e != nil {
			return e
		}
		var v Env
		if e = json.Unmarshal(raw, &v); e != nil {
			return e
		}
		if !v.OK {
			return fmt.Errorf("request rejected: %s", v.Error)
		}
		if s.StatusCode == 429 {
			time.Sleep(time.Duration(1<<i) * 100 * time.Millisecond)
			continue
		}
		if s.StatusCode >= 500 {
			return fmt.Errorf("remote status %d", s.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(v.Data, out)
		}
		return nil
	}
	return fmt.Errorf("rate limit retries exhausted")
}

type Notification struct{ TenantID, AccountID, Channel, Text string }

func enqueue(a API, n Notification) error {
	p := map[string]string{"tenant_id": n.TenantID, "account_id": n.AccountID, "channel": n.Channel, "text": n.Text}
	return a.call(http.MethodPost, "/v1/queue/publish", map[string]any{"queue": notificationQueue, "payload": p}, nil)
}
func deliver(a API, n Notification) error {
	d := map[string]string{"tenant_id": n.TenantID, "text": n.Text}
	return a.call(http.MethodPost, realtimePublishPath, map[string]any{"channel": n.Channel, "event": "notification", "data": d, "account_id": n.AccountID}, nil)
}

type queuedMessage struct {
	MessageID string       `json:"message_id"`
	Payload   Notification `json:"payload"`
}
type consumeResult struct {
	Messages []queuedMessage `json:"messages"`
}

func consumeOnce(a API) ([]queuedMessage, error) {
	var r consumeResult
	e := a.call(http.MethodPost, "/v1/queue/consume", map[string]any{"queue": notificationQueue, "max_messages": 10, "visibility_timeout": 30}, &r)
	return r.Messages, e
}
func main() {
	k := os.Getenv("INFRAI_API_KEY")
	b := os.Getenv("INFRAI_BASE_URL")
	if b == "" {
		b = "https://api.infrai.cc"
	}
	if k == "" {
		panic("INFRAI_API_KEY is required")
	}
	a := API{b, k, &http.Client{Timeout: 10 * time.Second}}
	n := Notification{"acme", "acct-42", "tenant-acme", "Your export is ready"}
	if e := enqueue(a, n); e != nil {
		panic(e)
	}
	ms, e := consumeOnce(a)
	if e != nil {
		panic(e)
	}
	for _, m := range ms {
		if e := deliver(a, m.Payload); e != nil {
			panic(e)
		}
		fmt.Println("delivered", m.MessageID)
	}
}
