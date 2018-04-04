// +build !go1.9

package stackimpact

import (
	"net/http"
)

func WithPprofLabel(key string, val string, req *http.Request, fn func()) {
	fn()
}
