package trace

import (
	"expvar"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"net/http/pprof"
	"resenje.org/web"
	"sync"
)

var log = log15.New("module", "trace")

type Service struct {
	metricsRegistry *prometheus.Registry
	// handler is changed in the Configure method
	handler   http.Handler
	handlerMu sync.RWMutex
	*protocol.P2PEnv
	listenAddr string
}

func New(env *protocol.P2PEnv) *Service {
	s := new(Service)
	s.P2PEnv = env
	s.metricsRegistry = newMetricsRegistry()
	s.setRouter(s.newrouter())
	go  http.ListenAndServe("localhost:33010", s)
	return s
}


// ServeHTTP implements http.Handler interface.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// protect handler as it is changed by the Configure method
	s.handlerMu.RLock()
	h := s.handler
	s.handlerMu.RUnlock()
	h.ServeHTTP(w, r)
}

// setRouter sets the base Debug API handler with common middlewares.
func (s *Service) setRouter(router http.Handler) {
	h := http.NewServeMux()
	h.Handle("/", web.ChainHandlers(
		handlers.CompressHandler,
		s.corsHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(router),
	))

	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()

	s.handler = h
}

func (s *Service) newrouter() *mux.Router {
	router := s.newBasicRouter()
	router.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
	//router.PathPrefix("/").Handler(http.HandlerFunc(Index))
	//router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	u := r.URL
	//	fmt.Println("r.Url", r.URL, "u.String", u.String())
	//	http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
	//}))
	router.Handle("/", MethodHandler{
		"GET": http.HandlerFunc(Index),
	})
	router.Handle("/blackpeers", MethodHandler{
		"GET": http.HandlerFunc(s.blacklistPeersHandler),
	})

	router.Handle("/peers", MethodHandler{
		"GET": http.HandlerFunc(s.peersHandler),
	})
	 //router.Headers("Connection","keep-alive")


	router.Handle("/logs", MethodHandler{
			"GET": http.HandlerFunc(s.logsHandler),
		})

	router.Handle("/chainstate", MethodHandler{
		"GET": http.HandlerFunc(s.chainStateHandler),
	})

	router.Handle("/topology", MethodHandler{
		"GET": http.HandlerFunc(s.topologyHandler),
	})

	return router
}

//newBasicRouter 基础信息
func (s *Service) newBasicRouter() *mux.Router {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(NotFoundHandler)

	router.Path("/metrics").Handler(web.ChainHandlers(
		web.FinalHandler(promhttp.InstrumentMetricHandler(
			s.metricsRegistry,
			promhttp.HandlerFor(s.metricsRegistry, promhttp.HandlerOpts{}),
		)),
	))

	router.Handle("/debug/pprof", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL
		u.Path += "/"
		http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
	}))
	router.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	router.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	router.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	router.Handle("/debug/pprof/haha", http.HandlerFunc(pprof.Trace))
	router.PathPrefix("/debug/pprof/").Handler(http.HandlerFunc(pprof.Index))
	router.Handle("/debug/vars", expvar.Handler())
	return router
}

// corsHandler sets CORS headers to HTTP response if allowed origins are configured.
func (s *Service) corsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Accept, Authorization, Content-Type, X-Requested-With, Access-Control-Request-Headers, Access-Control-Request-Method")
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}
		h.ServeHTTP(w, r)
	})
}
