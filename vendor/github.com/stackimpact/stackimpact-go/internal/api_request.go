package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"time"
)

type APIRequest struct {
	agent *Agent
}

func newAPIRequest(agent *Agent) *APIRequest {
	ar := &APIRequest{
		agent: agent,
	}

	return ar
}

func (ar *APIRequest) post(endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"runtime_type":    "go",
		"runtime_version": runtime.Version(),
		"agent_version":   AgentVersion,
		"app_name":        ar.agent.AppName,
		"app_version":     ar.agent.AppVersion,
		"app_environment": ar.agent.AppEnvironment,
		"host_name":       ar.agent.HostName,
		"process_id":      strconv.Itoa(os.Getpid()),
		"build_id":        ar.agent.buildId,
		"run_id":          ar.agent.runId,
		"run_ts":          ar.agent.runTs,
		"sent_at":         time.Now().Unix(),
		"payload":         payload,
	}

	reqBodyJson, _ := json.Marshal(reqBody)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(reqBodyJson)
	w.Close()

	u := ar.agent.DashboardAddress + "/agent/v1/" + endpoint
	req, err := http.NewRequest("POST", u, &buf)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(ar.agent.AgentKey, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	ar.agent.log("Posting API request to %v", u)

	var httpClient *http.Client
	if ar.agent.HTTPClient != nil {
		httpClient = ar.agent.HTTPClient
	} else if ar.agent.ProxyAddress != "" {
		proxyURL, err := url.Parse(ar.agent.ProxyAddress)
		if err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
			Timeout:   time.Second * 20,
		}
	} else {
		httpClient = &http.Client{
			Timeout: time.Second * 20,
		}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBodyJson, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Received %v: %v", res.StatusCode, string(resBodyJson)))
	} else {
		var resBody map[string]interface{}
		if err := json.Unmarshal(resBodyJson, &resBody); err != nil {
			return nil, errors.New(fmt.Sprintf("Cannot parse response body %v", string(resBodyJson)))
		} else {
			return resBody, nil
		}
	}
}
