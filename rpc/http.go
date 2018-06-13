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
		if !checkIpWhitelist(strings.Split(r.RemoteAddr, ":")[0]) {
			w.Write([]byte(fmt.Sprintf(`{"errcode":"-1","result":null,"msg":"The %s Address is not authorized!"}`, strings.Split(r.RemoteAddr, ":")[0])))
			return
		}
		if r.URL.Path == "/" {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				w.Write([]byte(`{"errcode":"-1","result":null,"msg":"Can't get request body!"}`))
				return
			}
			//Release local request
			if strings.Split(r.RemoteAddr, ":")[0] != "127.0.0.1" {
				client, err := parseJsonRpcParams(data)
				if err != nil {
					w.Write([]byte(fmt.Sprintf(`{"errcode":"-1","result":null,"msg":"The request content cannot be identified!"}`)))
					return
				}
				funcName := strings.Split(client.Method, ".")[len(strings.Split(client.Method, "."))-1]
				if !checkJrpcFuncWritelist(funcName) {
					w.Write([]byte(fmt.Sprintf(`{"errcode":"-1","result":null,"msg":"The %s method is not authorized!"}`, funcName)))
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

func (g *Grpcserver) Listen() {
	listener, err := net.Listen("tcp", rpcCfg.GetGrpcBindAddr())
	if err != nil {
		log.Crit("failed to listen:", "err", err)
		panic(err)
	}
	var opts []grpc.ServerOption
	//register interceptor
	var interceptor grpc.UnaryServerInterceptor
	interceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
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
func auth(ctx context.Context, info *grpc.UnaryServerInfo) error {
	getctx, ok := pr.FromContext(ctx)
	if ok {
		remoteaddr := strings.Split(getctx.Addr.String(), ":")[0]
		if remoteaddr == "127.0.0.1" {
			return nil
		}
		if !checkIpWhitelist(remoteaddr) {
			return fmt.Errorf("The %s Address is not authorized!", remoteaddr)
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
