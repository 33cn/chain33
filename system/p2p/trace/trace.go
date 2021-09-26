package trace

import (
	"encoding/hex"
	"errors"
	"expvar"
	"fmt"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/system/p2p/dht/protocol/peer"
	"github.com/33cn/chain33/types"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"resenje.org/web"
	"strings"
	"sync"
	"time"
)
var (
	log = log15.New("module", "trace")
	visterList =make(map[string]*types.Vister)
	visterLock sync.Mutex
	TraceService *Service
)



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
	s.setRouter(s.newrouter())
	go  http.ListenAndServe(s.SubConfig.Trace.ListenAddr, s)
	TraceService=s
	return s
}


func (s *Service)AddAuth(vister *types.Vister)(string,error){
	netMsg :=s.QueueClient.NewMessage("p2p", types.EventGetNetInfo, nil)
	s.QueueClient.Send(netMsg,false)
	resp,err:= s.QueueClient.Wait(netMsg)
	if err!=nil{
		return  "",err
	}

	netinfo, ok := resp.GetData().(*types.NodeNetInfo)
	if !ok{
		return "",errors.New("wrong data")
	}

	sport:=strings.Split(s.SubConfig.Trace.ListenAddr,":")

	duration,err:= peer.CaculateLifeTime(vister.GetLifetime())
	if err!=nil{
		return "",err
	}
	deadline:=time.Now().Add(duration)
	vister.Lifetime=deadline.String()

	var buf [8]byte
	rand.Read(buf[:])
	vister.Accesskey=hex.EncodeToString(buf[:])
		//fmt.Sprintf("http://%v:%v/logs?accesskey=%v",netinfo.GetExternaladdr(),sport[len(sport)-1],hex.EncodeToString(buf[:]))
	visterLock.Lock()
	defer visterLock.Unlock()
	if v,ok:=visterList[vister.GetRemoteIP()];ok{
		return v.Accesskey,nil
	}

	visterList[vister.GetRemoteIP()]=vister
	log.Info("AddAuth","remoteIP",vister.GetRemoteIP())
	//return vister.Accesskey,nil
	return fmt.Sprintf("http://%v:%v?accesskey=%v",netinfo.GetExternaladdr(),sport[len(sport)-1],vister.Accesskey),nil
}

func (s *Service)ShowVisters()*types.Visters{
	visterLock.Lock()
	defer visterLock.Unlock()
	now:=time.Now()
	var visters types.Visters
	for k,vister:=range visterList{
		//处理lifetime
		tm,_ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST",vister.GetLifetime())
		liftetime:= tm.Sub(now)/time.Second
		if liftetime <=0{
			delete(visterList,k)
			continue
		}

		vister.Lifetime=fmt.Sprintf("%v",liftetime)
		visters.Visters=append(visters.Visters,vister)
	}

	return &visters
}

func (s *Service) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	log.Info("remoteAddr","addr:",r.RemoteAddr)

	host,_,err := net.SplitHostPort(r.RemoteAddr)
	if err!=nil{
		return false
	}

	nip:= net.ParseIP(host)
	log.Info("checkAuth","isloopback",nip.IsLoopback(),"host:",host)
	if host == "localhost" || host == "127.0.0.1" || nip.IsLoopback(){
		//本地访问不需要权限
		return true
	}
	//chekc url param
	vls, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return false
	}
	log.Info("checkAuth","url param",vls.Get("accesskey"), "remoteIP",host,"visterList",visterList)
	key := vls.Get("accesskey")
	visterLock.Lock()
	defer visterLock.Unlock()
	if v,ok:=visterList[host];ok{

		if key == v.GetAccesskey(){
			log.Info("checkAuth success","key",key,"storeaccess key",v.GetAccesskey())
			return true
		}
		log.Info("checkAuth","key",key,"storeaccess key",v.GetAccesskey())
	}
	return false
}

// ServeHTTP implements http.Handler interface.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// protect handler as it is changed by the Configure method
	//CHECK AUTH
	if !s.checkAuth(w, r) {
		w.WriteHeader(403)
		BadRequest(w, "not authorized")
		return
	}

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
	router.PathPrefix("/").Handler(http.HandlerFunc(Index))
	/*router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL
		fmt.Println("r.Url", r.URL, "u.String", u.String())
		http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
	}))
	router.Handle("/index", MethodHandler{
		"GET": http.HandlerFunc(Index),
	})*/
	router.Handle("/blackpeers", MethodHandler{
		"GET": http.HandlerFunc(s.blacklistPeersHandler),
	})

	router.Handle("/peers", MethodHandler{
		"GET": http.HandlerFunc(s.peersHandler),
	})

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
