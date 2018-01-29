package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExpire(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	msg := map[string]interface{}{
		"a": 1,
	}

	agent.messageQueue.addMessage("test", msg)

	msg = map[string]interface{}{
		"a": 2,
	}

	agent.messageQueue.addMessage("test", msg)

	if len(agent.messageQueue.queue) != 2 {
		t.Errorf("Message len is not 2, but %v", len(agent.messageQueue.queue))
		return
	}

	agent.messageQueue.queue[0].addedAt = time.Now().Unix() - 20*60

	agent.messageQueue.expire()

	if len(agent.messageQueue.queue) != 1 {
		t.Errorf("Message len is not 1, but %v", len(agent.messageQueue.queue))
		return
	}
}

func TestFlush(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{}")
	}))
	defer server.Close()

	agent := NewAgent()
	agent.AgentKey = "key1"
	agent.AppName = "App1"
	agent.HostName = "Host1"
	agent.Debug = true
	agent.DashboardAddress = server.URL

	msg := map[string]interface{}{
		"a": 1,
	}
	agent.messageQueue.addMessage("test", msg)

	msg = map[string]interface{}{
		"a": 2,
	}
	agent.messageQueue.addMessage("test", msg)

	agent.messageQueue.flush()

	if len(agent.messageQueue.queue) > 0 {
		t.Errorf("Should have no messages, but has %v", len(agent.messageQueue.queue))
	}
}

func TestFlushFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "invalidjson")
	}))
	defer server.Close()

	agent := NewAgent()
	agent.AgentKey = "key1"
	agent.AppName = "App1"
	agent.HostName = "Host1"
	agent.Debug = true
	agent.DashboardAddress = server.URL

	msg := map[string]interface{}{
		"a": 1,
	}
	agent.messageQueue.addMessage("test", msg)

	msg = map[string]interface{}{
		"a": 2,
	}
	agent.messageQueue.addMessage("test", msg)

	agent.messageQueue.flush()

	if len(agent.messageQueue.queue) != 2 {
		t.Errorf("Should have 2 messages, but has %v", len(agent.messageQueue.queue))
	}
}
