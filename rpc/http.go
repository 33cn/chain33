package rpc

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/rs/cors"
	pb "gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"
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
			w.Write([]byte(`{"errcode":"-1","result":null,"msg":"reject"}`))
			return
		}
		if r.URL.Path == "/" {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				w.Write([]byte(`{"errcode":"-1","result":null,"msg":"wrong params"}`))
				return
			}
			//Release local request
			if strings.Split(r.RemoteAddr, ":")[0] != "127.0.0.1" {
				client, err := parseJsonRpcParams(data)
				if err != nil {
					w.Write([]byte(`{"errcode":"-1","result":null,"msg":"wrong params"}`))
					return
				}
				if !checkJrpcFuncWritelist(client.Method) {
					w.Write([]byte(`{"errcode":"-1","result":null,"msg":"reject"}`))
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
	s := grpc.NewServer()
	pb.RegisterGrpcserviceServer(s, &g.grpc)
	s.Serve(listener)

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
