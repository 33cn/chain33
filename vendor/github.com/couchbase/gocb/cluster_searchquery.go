package gocb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"gopkg.in/couchbaselabs/jsonx.v1"
)

// SearchResultLocation holds the location of a hit in a list of search results.
type SearchResultLocation struct {
	Position       int    `json:"position,omitempty"`
	Start          int    `json:"start,omitempty"`
	End            int    `json:"end,omitempty"`
	ArrayPositions []uint `json:"array_positions,omitempty"`
}

// SearchResultHit holds a single hit in a list of search results.
type SearchResultHit struct {
	Index       string                                       `json:"index,omitempty"`
	Id          string                                       `json:"id,omitempty"`
	Score       float64                                      `json:"score,omitempty"`
	Explanation map[string]interface{}                       `json:"explanation,omitempty"`
	Locations   map[string]map[string][]SearchResultLocation `json:"locations,omitempty"`
	Fragments   map[string][]string                          `json:"fragments,omitempty"`
	Fields      map[string]string                            `json:"fields,omitempty"`
}

// SearchResultTermFacet holds the results of a term facet in search results.
type SearchResultTermFacet struct {
	Term  string `json:"term,omitempty"`
	Count int    `json:"count,omitempty"`
}

// SearchResultNumericFacet holds the results of a numeric facet in search results.
type SearchResultNumericFacet struct {
	Name  string  `json:"name,omitempty"`
	Min   float64 `json:"min,omitempty"`
	Max   float64 `json:"max,omitempty"`
	Count int     `json:"count,omitempty"`
}

// SearchResultDateFacet holds the results of a date facet in search results.
type SearchResultDateFacet struct {
	Name  string `json:"name,omitempty"`
	Min   string `json:"min,omitempty"`
	Max   string `json:"max,omitempty"`
	Count int    `json:"count,omitempty"`
}

// SearchResultFacet holds the results of a specified facet in search results.
type SearchResultFacet struct {
	Field         string                     `json:"field,omitempty"`
	Total         int                        `json:"total,omitempty"`
	Missing       int                        `json:"missing,omitempty"`
	Other         int                        `json:"other,omitempty"`
	Terms         []SearchResultTermFacet    `json:"terms,omitempty"`
	NumericRanges []SearchResultNumericFacet `json:"numeric_ranges,omitempty"`
	DateRanges    []SearchResultDateFacet    `json:"date_ranges,omitempty"`
}

// SearchResultStatus holds the status information for an executed search query.
type SearchResultStatus struct {
	Total      int `json:"total,omitempty"`
	Failed     int `json:"failed,omitempty"`
	Successful int `json:"successful,omitempty"`
}

// SearchResults allows access to the results of a search query.
type SearchResults interface {
	Status() SearchResultStatus
	Errors() []string
	TotalHits() int
	Hits() []SearchResultHit
	Facets() map[string]SearchResultFacet
	Took() time.Duration
	MaxScore() float64
}

type searchResponse struct {
	Status    SearchResultStatus           `json:"status,omitempty"`
	Errors    []string                     `json:"errors,omitempty"`
	TotalHits int                          `json:"total_hits,omitempty"`
	Hits      []SearchResultHit            `json:"hits,omitempty"`
	Facets    map[string]SearchResultFacet `json:"facets,omitempty"`
	Took      uint                         `json:"took,omitempty"`
	MaxScore  float64                      `json:"max_score,omitempty"`
}

type searchResults struct {
	data *searchResponse
}

func (r searchResults) Status() SearchResultStatus {
	return r.data.Status
}
func (r searchResults) Errors() []string {
	return r.data.Errors
}
func (r searchResults) TotalHits() int {
	return r.data.TotalHits
}
func (r searchResults) Hits() []SearchResultHit {
	return r.data.Hits
}
func (r searchResults) Facets() map[string]SearchResultFacet {
	return r.data.Facets
}
func (r searchResults) Took() time.Duration {
	return time.Duration(r.data.Took) / time.Nanosecond
}
func (r searchResults) MaxScore() float64 {
	return r.data.MaxScore
}

// Performs a spatial query and returns a list of rows or an error.
func (c *Cluster) doSearchQuery(tracectx opentracing.SpanContext, b *Bucket, q *SearchQuery) (SearchResults, error) {
	var err error
	var ftsEp string
	var timeout time.Duration
	var client *http.Client
	var creds []UserPassPair

	if b != nil {
		ftsEp, err = b.getFtsEp()
		if err != nil {
			return nil, err
		}

		if b.ftsTimeout < c.ftsTimeout {
			timeout = b.ftsTimeout
		} else {
			timeout = c.ftsTimeout
		}
		client = b.client.HttpClient()
		if c.auth != nil {
			creds, err = c.auth.Credentials(AuthCredsRequest{
				Service:  FtsService,
				Endpoint: ftsEp,
				Bucket:   b.name,
			})
			if err != nil {
				return nil, err
			}
		} else {
			creds = []UserPassPair{
				{
					Username: b.name,
					Password: b.password,
				},
			}
		}
	} else {
		if c.auth == nil {
			panic("Cannot perform cluster level queries without Cluster Authenticator.")
		}

		tmpB, err := c.randomBucket()
		if err != nil {
			return nil, err
		}

		ftsEp, err = tmpB.getFtsEp()
		if err != nil {
			return nil, err
		}

		timeout = c.ftsTimeout
		client = tmpB.client.HttpClient()

		creds, err = c.auth.Credentials(AuthCredsRequest{
			Service:  FtsService,
			Endpoint: ftsEp,
		})
		if err != nil {
			return nil, err
		}
	}

	qIndexName := q.indexName()
	qBytes, err := json.Marshal(q.queryData())
	if err != nil {
		return nil, err
	}

	var queryData jsonx.DelayedObject
	err = json.Unmarshal(qBytes, &queryData)
	if err != nil {
		return nil, err
	}

	var ctlData jsonx.DelayedObject
	if queryData.Has("ctl") {
		err = queryData.Get("ctl", &ctlData)
		if err != nil {
			return nil, err
		}
	}

	qTimeout := jsonMillisecondDuration(timeout)
	if ctlData.Has("timeout") {
		err := ctlData.Get("timeout", &qTimeout)
		if err != nil {
			return nil, err
		}
		if qTimeout <= 0 || time.Duration(qTimeout) > timeout {
			qTimeout = jsonMillisecondDuration(timeout)
		}
	}
	err = ctlData.Set("timeout", qTimeout)
	if err != nil {
		return nil, err
	}

	err = queryData.Set("ctl", ctlData)
	if err != nil {
		return nil, err
	}

	if len(creds) > 1 {
		err = queryData.Set("creds", creds)
		if err != nil {
			return nil, err
		}
	}

	qBytes, err = json.Marshal(queryData)
	if err != nil {
		return nil, err
	}

	reqUri := fmt.Sprintf("%s/api/index/%s/query", ftsEp, qIndexName)

	req, err := http.NewRequest("POST", reqUri, bytes.NewBuffer(qBytes))
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

	ftsResp := searchResponse{}
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&ftsResp)
	if err != nil {
		strace.Finish()
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	strace.Finish()

	if resp.StatusCode != 200 {
		return nil, &viewError{
			Message: "HTTP Error",
			Reason:  fmt.Sprintf("Status code was %d.", resp.StatusCode),
		}
	}

	return searchResults{
		data: &ftsResp,
	}, nil
}

// ExecuteSearchQuery performs a n1ql query and returns a list of rows or an error.
func (c *Cluster) ExecuteSearchQuery(q *SearchQuery) (SearchResults, error) {
	span := c.agentConfig.Tracer.StartSpan("ExecuteSearchQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "fts"})
	defer span.Finish()

	return c.doSearchQuery(span.Context(), nil, q)
}
