package gocb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
)

type analyticsError struct {
	Code    uint32 `json:"code"`
	Message string `json:"msg"`
}

func (e *analyticsError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

type analyticsResponse struct {
	RequestId       string            `json:"requestID"`
	ClientContextId string            `json:"clientContextID"`
	Results         []json.RawMessage `json:"results,omitempty"`
	Errors          []analyticsError  `json:"errors,omitempty"`
	Status          string            `json:"status"`
}

type analyticsMultiError []analyticsError

func (e *analyticsMultiError) Error() string {
	return (*e)[0].Error()
}

func (e *analyticsMultiError) Code() uint32 {
	return (*e)[0].Code
}

// AnalyticsResults allows access to the results of a Analytics query.
type AnalyticsResults interface {
	One(valuePtr interface{}) error
	Next(valuePtr interface{}) bool
	NextBytes() []byte
	Close() error

	RequestId() string
	ClientContextId() string
}

type analyticsResults struct {
	closed          bool
	index           int
	rows            []json.RawMessage
	err             error
	requestId       string
	clientContextId string
}

func (r *analyticsResults) Next(valuePtr interface{}) bool {
	if r.err != nil {
		return false
	}

	row := r.NextBytes()
	if row == nil {
		return false
	}

	r.err = json.Unmarshal(row, valuePtr)
	if r.err != nil {
		return false
	}

	return true
}

func (r *analyticsResults) NextBytes() []byte {
	if r.err != nil {
		return nil
	}

	if r.index+1 >= len(r.rows) {
		r.closed = true
		return nil
	}
	r.index++

	return r.rows[r.index]
}

func (r *analyticsResults) Close() error {
	r.closed = true
	return r.err
}

func (r *analyticsResults) One(valuePtr interface{}) error {
	if !r.Next(valuePtr) {
		err := r.Close()
		if err != nil {
			return err
		}
		return ErrNoResults
	}

	// Ignore any errors occurring after we already have our result
	err := r.Close()
	if err != nil {
		// Return no error as we got the one result already.
		return nil
	}

	return nil
}

func (r *analyticsResults) RequestId() string {
	if !r.closed {
		panic("Result must be closed before accessing meta-data")
	}

	return r.requestId
}

func (r *analyticsResults) ClientContextId() string {
	if !r.closed {
		panic("Result must be closed before accessing meta-data")
	}

	return r.clientContextId
}

func (c *Cluster) executeAnalyticsQuery(tracectx opentracing.SpanContext, analyticsEp string, opts map[string]interface{}, creds []UserPassPair, timeout time.Duration, client *http.Client) (AnalyticsResults, error) {
	reqUri := fmt.Sprintf("%s/query/service", analyticsEp)

	tmostr, castok := opts["timeout"].(string)
	if castok {
		var err error
		timeout, err = time.ParseDuration(tmostr)
		if err != nil {
			return nil, err
		}
	} else {
		// Set the timeout string to its default variant
		opts["timeout"] = timeout.String()
	}

	if len(creds) > 1 {
		opts["creds"] = creds
	}

	reqJson, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqUri, bytes.NewBuffer(reqJson))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if len(creds) == 1 {
		req.SetBasicAuth(creds[0].Username, creds[0].Password)
	}

	dtrace := c.agentConfig.Tracer.StartSpan("dispatch",
		opentracing.ChildOf(tracectx))

	resp, err := doHttpWithTimeout(client, req, timeout)
	if err != nil {
		dtrace.Finish()
		return nil, err
	}

	dtrace.Finish()

	strace := c.agentConfig.Tracer.StartSpan("streaming",
		opentracing.ChildOf(tracectx))

	analyticsResp := analyticsResponse{}
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&analyticsResp)
	if err != nil {
		strace.Finish()
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	strace.Finish()

	if len(analyticsResp.Errors) > 0 {
		return nil, (*analyticsMultiError)(&analyticsResp.Errors)
	}

	if resp.StatusCode != 200 {
		return nil, &viewError{
			Message: "HTTP Error",
			Reason:  fmt.Sprintf("Status code was %d.", resp.StatusCode),
		}
	}

	return &analyticsResults{
		requestId:       analyticsResp.RequestId,
		clientContextId: analyticsResp.ClientContextId,
		index:           -1,
		rows:            analyticsResp.Results,
	}, nil
}

// EnableAnalytics allows you to specify Analytics hosts to perform queries against.
//
// Experimental: This API is only needed temporarily until full integration of the
// Analytics service into Couchbase Server has been completed.
func (c *Cluster) EnableAnalytics(hosts []string) {
	c.analyticsHosts = make([]string, len(hosts))

	for i, host := range hosts {
		host := strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")

		if c.agentConfig.TlsConfig == nil {
			host = "http://" + host
		} else {
			host = "https://" + host
		}

		c.analyticsHosts[i] = host
	}
}

// Performs a spatial query and returns a list of rows or an error.
func (c *Cluster) doAnalyticsQuery(tracectx opentracing.SpanContext, q *AnalyticsQuery) (AnalyticsResults, error) {
	numHosts := len(c.analyticsHosts)
	if numHosts == 0 {
		return nil, fmt.Errorf("must specify analytics hosts with EnableAnalytics first")
	}

	analyticsEp := c.analyticsHosts[rand.Intn(numHosts)]

	if c.auth == nil {
		panic("Cannot perform cluster level queries without Cluster Authenticator.")
	}

	creds, err := c.auth.Credentials(AuthCredsRequest{
		Service:  CbasService,
		Endpoint: analyticsEp,
	})
	if err != nil {
		return nil, err
	}

	return c.executeAnalyticsQuery(tracectx, analyticsEp, q.options, creds, c.analyticsTimeout, c.httpCli)
}

// ExecuteAnalyticsQuery performs an analytics query and returns a list of rows or an error.
//
// Experimental: This API is subject to change at any time.
func (c *Cluster) ExecuteAnalyticsQuery(q *AnalyticsQuery) (AnalyticsResults, error) {
	span := c.agentConfig.Tracer.StartSpan("ExecuteAnalyticsQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "analytics"})
	defer span.Finish()

	return c.doAnalyticsQuery(span.Context(), q)
}
