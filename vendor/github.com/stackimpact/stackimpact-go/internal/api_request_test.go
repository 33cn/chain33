package internal

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gr, _ := gzip.NewReader(r.Body)
		bodyJson, _ := ioutil.ReadAll(gr)
		gr.Close()

		var body map[string]interface{}
		if err := json.Unmarshal(bodyJson, &body); err != nil {
			t.Error(err)
		} else {
			if prop, ok := body["runtime_type"]; !ok || prop.(string) != "go" {
				t.Errorf("Invalid or missing property: %v", "runtime_type")
			}

			if prop, ok := body["runtime_version"]; !ok || prop.(string) != runtime.Version() {
				t.Errorf("Invalid or missing property: %v", "runtime_version")
			}

			if prop, ok := body["agent_version"]; !ok || prop.(string) != AgentVersion {
				t.Errorf("Invalid or missing property: %v", "agent_version")
			}

			if prop, ok := body["app_name"]; !ok || prop.(string) != "App1" {
				t.Errorf("Invalid or missing property: %v", "app_name")
			}

			if prop, ok := body["host_name"]; !ok || prop.(string) != "Host1" {
				t.Errorf("Invalid or missing property: %v", "host_name")
			}

			if _, ok := body["run_id"]; !ok {
				t.Errorf("Invalid or missing property: %v", "run_id")
			}

			if prop, ok := body["run_ts"]; !ok || prop.(float64) < float64(time.Now().Unix()-60) {
				t.Errorf("Invalid or missing property: %v", "run_ts")
			}

			if prop, ok := body["sent_at"]; !ok || prop.(float64) < float64(time.Now().Unix()-60) {
				t.Errorf("Invalid or missing property: %v", "sent_at")
			}

			if _, ok := body["payload"]; !ok {
				t.Errorf("Invalid or missing property: %v", "payload")
			} else {
				payload := body["payload"].(map[string]interface{})

				if payload["a"].(float64) != 1 {
					t.Error(fmt.Sprintf("Invalid payload received: %v", payload))
				}
			}
		}

		fmt.Fprintf(w, "{}")
	}))
	defer server.Close()

	agent := NewAgent()
	agent.AppName = "App1"
	agent.HostName = "Host1"
	agent.Debug = true
	agent.DashboardAddress = server.URL

	p := map[string]interface{}{
		"a": 1,
	}
	agent.apiRequest.post("test", p)
}
