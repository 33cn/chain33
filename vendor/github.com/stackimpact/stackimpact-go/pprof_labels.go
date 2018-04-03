// +build go1.9

package stackimpact

import (
	"context"
	"net/http"
	"runtime/pprof"
)

func WithPprofLabel(key string, val string, req *http.Request, fn func()) {
	labelSet := pprof.Labels(key, val)
	pprof.Do(req.Context(), labelSet, func(ctx context.Context) {
		fn()
	})
}
