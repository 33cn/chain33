package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{\"agent_enabled\":\"yes\"}")
	}))
	defer server.Close()

	agent := NewAgent()
	agent.AgentKey = "key1"
	agent.AppName = "App1"
	agent.HostName = "Host1"
	agent.Debug = true
	agent.DashboardAddress = server.URL

	agent.config.setAgentEnabled(false)

	agent.configLoader.load()

	if !agent.config.isAgentEnabled() {
		t.Errorf("Config loading wasn't successful")
	}
}
