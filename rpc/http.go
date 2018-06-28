package rpc

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"

	"github.com/rs/cors"
	pb "gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pr "google.golang.org/grpc/peer"
)

// adapt HTTP connection to ReadWriteCloser
type HTTPConn struct {
	r   *http.Request
	in  io.Reader
	out io.Writer
}

func (c *HTTPConn) Read(p []byte) (n int, err error) { return c.in.Read(p) }

func (c *HTTPConn) Write(d []byte) (n int, err error) { //添加支持gzip 发送

	if strings.Contains(c.r.Header.Get("Accept-Encoding"), "gzip") {
		gw := gzip.NewWriter(c.out)
		defer gw.Close()
		return gw.Write(d)
	}
	return c.out.Write(d)
}

func (c *HTTPConn) Close() error { return nil }

func (j *JSONRPCServer) Listen() {
	listener, err := net.Listen("tcp", rpcCfg.GetJrpcBindAddr())
	if err != nil {
		log.Crit("listen:", "err", err)
		panic(err)
	}
	server := rpc.NewServer()

	server.Register(&j.jrpc)
	co := cors.New(cors.Options{})

	// Insert the middleware
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("JSONRPCServer", "RemoteAddr", r.RemoteAddr)
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			writeError(w, r, 0, fmt.Sprintf(`The %s Address is not authorized!`, ip))
			return
		}

		if !checkIpWhitelist(ip) {
			writeError(w, r, 0, fmt.Sprintf(`The %s Address is not authorized!`, ip))
			return
		}
		if r.URL.Path == "/" {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				writeError(w, r, 0, "Can't get request body!")
				return
			}
			//Release local request
			ipaddr := net.ParseIP(ip)
			if !ipaddr.IsLoopback() {
				client, err := parseJsonRpcParams(data)
				if err != nil {
					writeError(w, r, 0, fmt.Sprintf(`parse request err %s`, err.Error()))
					return
				}
				funcName := strings.Split(client.Method, ".")[len(strings.Split(client.Method, "."))-1]
				if !checkJrpcFuncWritelist(funcName) {
					writeError(w, r, client.Id, fmt.Sprintf(`The %s method is not authorized!`, funcName))
					return
				}
			}
			serverCodec := jsonrpc.NewServerCodec(&HTTPConn{in: ioutil.NopCloser(bytes.NewReader(data)), out: w, r: r})
			w.Header().Set("Content-type", "application/json")
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				w.Header().Set("Content-Encoding", "gzip")
			}
			w.WriteHeader(200)
			err = server.ServeRequest(serverCodec)
			if err != nil {
				log.Debug("Error while serving JSON request: %v", err)
				return
			}
		}
	})

	handler = co.Handler(handler)
	http.Serve(listener, handler)
}

type serverResponse struct {
	Id     uint64      `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

func writeError(w http.ResponseWriter, r *http.Request, id uint64, errstr string) {
	w.Header().Set("Content-type", "application/json")
	//错误的请求也返回 200
	w.WriteHeader(200)
	resp, err := json.Marshal(&serverResponse{id, nil, errstr})
	if err != nil {
		log.Debug("json marshal error, nerver happen")
		return
	}
	w.Write(resp)
}

func (g *Grpcserver) Listen() {
	listener, err := net.Listen("tcp", rpcCfg.GetGrpcBindAddr())
	if err != nil {
		log.Crit("failed to listen:", "err", err)
		panic(err)
	}
	var opts []grpc.ServerOption
	//register interceptor
	//var interceptor grpc.UnaryServerInterceptor
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if err := auth(ctx, info); err != nil {
			return nil, err
		}
		// Continue processing the request
		return handler(ctx, req)
	}
	opts = append(opts, grpc.UnaryInterceptor(interceptor))
	s := grpc.NewServer(opts...)
	pb.RegisterGrpcserviceServer(s, &g.grpc)
	s.Serve(listener)

}

func isLoopBackAddr(addr net.Addr) bool {
	if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.IsLoopback() {
		return true
	}
	return false
}

func auth(ctx context.Context, info *grpc.UnaryServerInfo) error {
	getctx, ok := pr.FromContext(ctx)
	if ok {
		if isLoopBackAddr(getctx.Addr) {
			return nil
		}
		//remoteaddr := strings.Split(getctx.Addr.String(), ":")[0]
		ip, _, err := net.SplitHostPort(getctx.Addr.String())
		if err != nil {
			return fmt.Errorf("The %s Address is not authorized!", ip)
		}

		if !checkIpWhitelist(ip) {
			return fmt.Errorf("The %s Address is not authorized!", ip)
		}

		funcName := strings.Split(info.FullMethod, "/")[len(strings.Split(info.FullMethod, "/"))-1]
		if !checkGrpcFuncWritelist(funcName) {
			return fmt.Errorf("The %s method is not authorized!", funcName)
		}
		return nil
	}
	return fmt.Errorf("Can't get remote ip!")
}

func parseJsonRpcParams(data []byte) (clientRequest, error) {
	var req clientRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		return req, err
	}
	log.Debug("request method: %v", req.Method)
	return req, nil
}
