package rpc

import (
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"

	"github.com/rs/cors"
	pb "gitlab.33.cn/chain33/chain33/types"

	"google.golang.org/grpc"
)

// adapt HTTP connection to ReadWriteCloser
type HTTPConn struct {
	in  io.Reader
	out io.Writer
}

func (c *HTTPConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HTTPConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }
func (c *HTTPConn) Close() error                      { return nil }

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

		if checkWhitlist(strings.Split(r.RemoteAddr, ":")[0]) == false {
			w.Write([]byte(`{"errcode":"-1","result":null,"msg":"reject"}`))
			return
		}

		if r.URL.Path == "/" {
			serverCodec := jsonrpc.NewServerCodec(&HTTPConn{in: r.Body, out: w})
			w.Header().Set("Content-type", "application/json")
			w.WriteHeader(200)
			err := server.ServeRequest(serverCodec)
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
