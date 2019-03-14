// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	"net/rpc/jsonrpc"
	"strings"

	"github.com/rs/cors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pr "google.golang.org/grpc/peer"
)

// HTTPConn adapt HTTP connection to ReadWriteCloser
type HTTPConn struct {
	r   *http.Request
	in  io.Reader
	out io.Writer
}

// Read rewrite the read of http
func (c *HTTPConn) Read(p []byte) (n int, err error) { return c.in.Read(p) }

// Write rewrite the write of http
func (c *HTTPConn) Write(d []byte) (n int, err error) { //添加支持gzip 发送

	if strings.Contains(c.r.Header.Get("Accept-Encoding"), "gzip") {
		gw := gzip.NewWriter(c.out)
		defer gw.Close()
		return gw.Write(d)
	}
	return c.out.Write(d)
}

// Close rewrite the close of http
func (c *HTTPConn) Close() error { return nil }

// Listen jsonsever listen
func (j *JSONRPCServer) Listen() (int, error) {
	listener, err := net.Listen("tcp", rpcCfg.JrpcBindAddr)
	if err != nil {
		return 0, err
	}
	j.l = listener
	co := cors.New(cors.Options{})

	// Insert the middleware
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("JSONRPCServer", "RemoteAddr", r.RemoteAddr)
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			writeError(w, r, 0, fmt.Sprintf(`The %s Address is not authorized!`, ip))
			return
		}

		if !checkIPWhitelist(ip) {
			writeError(w, r, 0, fmt.Sprintf(`The %s Address is not authorized!`, ip))
			return
		}
		if r.URL.Path == "/" {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				writeError(w, r, 0, "Can't get request body!")
				return
			}
			//格式做一个检查
			client, err := parseJSONRpcParams(data)
			errstr := "nil"
			if err != nil {
				errstr = err.Error()
			}
			funcName := strings.Split(client.Method, ".")[len(strings.Split(client.Method, "."))-1]
			if !checkFilterPrintFuncBlacklist(funcName) {
				log.Debug("JSONRPCServer", "request", string(data), "err", errstr)
			}
			if err != nil {
				writeError(w, r, 0, fmt.Sprintf(`parse request err %s`, err.Error()))
				return
			}
			//Release local request
			ipaddr := net.ParseIP(ip)
			if !ipaddr.IsLoopback() {
				//funcName := strings.Split(client.Method, ".")[len(strings.Split(client.Method, "."))-1]
				if checkJrpcFuncBlacklist(funcName) || !checkJrpcFuncWhitelist(funcName) {
					writeError(w, r, client.ID, fmt.Sprintf(`The %s method is not authorized!`, funcName))
					return
				}
			}
			serverCodec := jsonrpc.NewServerCodec(&HTTPConn{in: ioutil.NopCloser(bytes.NewReader(data)), out: w, r: r})
			w.Header().Set("Content-type", "application/json")
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				w.Header().Set("Content-Encoding", "gzip")
			}
			w.WriteHeader(200)
			err = j.s.ServeRequest(serverCodec)
			if err != nil {
				log.Debug("Error while serving JSON request: %v", err)
				return
			}
		}
	})

	handler = co.Handler(handler)
	if !rpcCfg.EnableTLS {
		go http.Serve(listener, handler)
	} else {
		go http.ServeTLS(listener, handler, rpcCfg.CertFile, rpcCfg.KeyFile)
	}
	return listener.Addr().(*net.TCPAddr).Port, nil
}

type serverResponse struct {
	ID     uint64      `json:"id"`
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
	_, err = w.Write(resp)
	if err != nil {
		log.Debug("Write", "err", err)
		return
	}
}

// Listen grpcserver listen
func (g *Grpcserver) Listen() (int, error) {
	listener, err := net.Listen("tcp", rpcCfg.GrpcBindAddr)
	if err != nil {
		return 0, err
	}
	g.l = listener
	go g.s.Serve(listener)
	return listener.Addr().(*net.TCPAddr).Port, nil
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
			return fmt.Errorf("the %s Address is not authorized", ip)
		}

		if !checkIPWhitelist(ip) {
			return fmt.Errorf("the %s Address is not authorized", ip)
		}

		funcName := strings.Split(info.FullMethod, "/")[len(strings.Split(info.FullMethod, "/"))-1]
		if checkGrpcFuncBlacklist(funcName) || !checkGrpcFuncWhitelist(funcName) {
			return fmt.Errorf("the %s method is not authorized", funcName)
		}
		return nil
	}
	return fmt.Errorf("can't get remote ip")
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	ID     uint64         `json:"id"`
}

func parseJSONRpcParams(data []byte) (*clientRequest, error) {
	var req clientRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}
