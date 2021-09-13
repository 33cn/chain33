package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"resenje.org/web"
)

var (
	// DefaultContentTypeHeader is the value of if "Content-Type" header
	// in HTTP response.
	DefaultContentTypeHeader = "application/json; charset=utf-8"
	// EscapeHTML specifies whether problematic HTML characters
	// should be escaped inside JSON quoted strings.
	EscapeHTML = false
)

type MethodHandler map[string]http.Handler

func (h MethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	web.HandleMethods(h, `{"message":"Method Not Allowed","code":405}`, DefaultContentTypeHeader, w, r)
}

// NewMaxBodyBytesHandler is an http middleware constructor that limits the
// maximal number of bytes that can be read from the request body. When a body
// is read, the error can be handled with a helper function HandleBodyReadError
// in order to respond with Request Entity Too Large response.
// See TestNewMaxBodyBytesHandler as an example.
func NewMaxBodyBytesHandler(limit int64) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				RequestEntityTooLarge(w, nil)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			h.ServeHTTP(w, r)
		})
	}
}

// HandleBodyReadError checks for particular errors and writes appropriate
// response accordingly. If no known error is found, no response is written and
// the function returns false.
func HandleBodyReadError(err error, w http.ResponseWriter) (responded bool) {
	if err == nil {
		return false
	}
	// http.MaxBytesReader returns an unexported error,
	// this is the only way to detect it
	if err.Error() == "http: request body too large" {
		RequestEntityTooLarge(w, nil)
		return true
	}
	return false
}

func OK(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusOK, response)
}

// BadRequest writes a response with status code 400.
func BadRequest(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusBadRequest, response)
}

// RequestEntityTooLarge writes a response with status code 413.
func RequestEntityTooLarge(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusRequestEntityTooLarge, response)
}

func NotFoundHandler(w http.ResponseWriter, _ *http.Request) {
	NotFound(w, nil)
}

// NotFound writes a response with status code 404.
func NotFound(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNotFound, response)
}

type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// Respond writes a JSON-encoded body to http.ResponseWriter.
func Respond(w http.ResponseWriter, statusCode int, response interface{}) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	if response == nil {
		response = &StatusResponse{
			Message: http.StatusText(statusCode),
			Code:    statusCode,
		}
	} else {
		switch message := response.(type) {
		case string:
			response = &StatusResponse{
				Message: message,
				Code:    statusCode,
			}
		case error:
			response = &StatusResponse{
				Message: message.Error(),
				Code:    statusCode,
			}
		case interface {
			String() string
		}:
			response = &StatusResponse{
				Message: message.String(),
				Code:    statusCode,
			}
		}
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(EscapeHTML)
	if err := enc.Encode(response); err != nil {
		panic(err)
	}
	if DefaultContentTypeHeader != "" {
		w.Header().Set("Content-Type", DefaultContentTypeHeader)
	}
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, b.String())
}
