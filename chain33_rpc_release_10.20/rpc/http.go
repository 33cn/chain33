package rpc

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"

	pb "code.aliyun.com/chain33/chain33/types"

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
	server := rpc.NewServer()
	server.Register(&JRpcRequest{jserver: jrpc})
	listener, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/" {
			serverCodec := jsonrpc.NewServerCodec(&HttpConn{in: r.Body, out: w})
			w.Header().Set("Content-type", "application/json")
			w.WriteHeader(200)
			err := server.ServeRequest(serverCodec)
			if err != nil {
				log.Printf("Error while serving JSON request: %v", err)
				return
			}
		}

	}))

}

func (grpcx *grpcServer) CreateServer(addr string) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGrpcserviceServer(s, &Grpc{gserver: grpcx})
	go s.Serve(listener)

}
