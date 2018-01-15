package rpc

import (
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"

	pb "code.aliyun.com/chain33/chain33/types"
	"github.com/rs/cors"

	"google.golang.org/grpc"
)

// adapt HTTP connection to ReadWriteCloser
type HttpConn struct {
	in  io.Reader
	out io.Writer
}

func (c *HttpConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HttpConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }
func (c *HttpConn) Close() error                      { return nil }

func (jrpc *jsonrpcServer) CreateServer(addr string) {
	listener, e := net.Listen("tcp", addr)
	if e != nil {
		log.Crit("listen:", "err", e)
		panic(e)
	}
	jrpc.listener = listener
	server := rpc.NewServer()
	var chain33 Chain33
	chain33.cli = NewClient("channel", "")
	chain33.cli.SetQueue(jrpc.q)
	chain33.jserver = jrpc
	server.Register(&chain33)
	c := cors.New(cors.Options{})

	// Insert the middleware
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			serverCodec := jsonrpc.NewServerCodec(&HttpConn{in: r.Body, out: w})
			w.Header().Set("Content-type", "application/json")
			w.WriteHeader(200)
			err := server.ServeRequest(serverCodec)
			if err != nil {
				log.Debug("Error while serving JSON request: %v", err)
				return
			}
		}
	})
	handler = c.Handler(handler)
	go http.Serve(listener, handler)
}

func (jrpc *jsonrpcServer) Close() {
	jrpc.listener.Close()
}

func (grpcx *grpcServer) CreateServer(addr string) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Crit("failed to listen:", "err", err)
		panic(err)
	}
	grpcx.listener = listener
	s := grpc.NewServer()
	var grpc Grpc
	grpc.cli = NewClient("channel", "")
	grpc.cli.SetQueue(grpcx.q)
	grpc.gserver = grpcx
	pb.RegisterGrpcserviceServer(s, &grpc)
	go s.Serve(listener)

}
