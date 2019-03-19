// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"fmt"
	"net"
	"sync"
	"time"

	pb "github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	pr "google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
)

// Listener the actions
type Listener interface {
	Close()
	Start()
}

// Start listener start
func (l *listener) Start() {
	l.p2pserver.Start()
	go l.server.Serve(l.netlistener)

}

// Close listener close
func (l *listener) Close() {
	err := l.netlistener.Close()
	if err != nil {
		log.Error("Close", "netlistener.Close() err", err)
	}
	go l.server.Stop()
	l.p2pserver.Close()
	log.Info("stop", "listener", "close")

}

type listener struct {
	server      *grpc.Server
	nodeInfo    *NodeInfo
	p2pserver   *P2pserver
	node        *Node
	netlistener net.Listener
}

// NewListener produce a listener object
func NewListener(protocol string, node *Node) Listener {
	log.Debug("NewListener", "localPort", node.listenPort)
	l, err := net.Listen(protocol, fmt.Sprintf(":%v", node.listenPort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}

	dl := &listener{
		nodeInfo:    node.nodeInfo,
		node:        node,
		netlistener: l,
	}

	pServer := NewP2pServer()
	pServer.node = dl.node

	//一元拦截器 接口调用之前进行校验拦截
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		//checkAuth
		getctx, ok := pr.FromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("")
		}
		ip, _, err := net.SplitHostPort(getctx.Addr.String())
		if err != nil {
			return nil, err
		}
		if pServer.node.nodeInfo.blacklist.Has(ip) {
			return nil, fmt.Errorf("blacklist %v no authorized", ip)
		}

		if !auth(ip) {
			log.Error("interceptor", "auth faild", ip)
			//把相应的IP地址加入黑名单中
			pServer.node.nodeInfo.blacklist.Add(ip, int64(3600))
			return nil, fmt.Errorf("auth faild %v  no authorized", ip)

		}
		// Continue processing the request
		return handler(ctx, req)
	}
	//流拦截器
	interceptorStream := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		getctx, ok := pr.FromContext(ss.Context())
		if !ok {
			log.Error("interceptorStream", "FromContext error", "")
			return fmt.Errorf("stream Context err")
		}
		ip, _, err := net.SplitHostPort(getctx.Addr.String())
		if err != nil {
			return err
		}
		if pServer.node.nodeInfo.blacklist.Has(ip) {
			return fmt.Errorf("blacklist %v  no authorized", ip)
		}

		if !auth(ip) {
			log.Error("interceptorStream", "auth faild", ip)
			//把相应的IP地址加入黑名单中
			pServer.node.nodeInfo.blacklist.Add(ip, int64(3600))
			return fmt.Errorf("auth faild  %v  no authorized", ip)
		}
		return handler(srv, ss)
	}
	var opts []grpc.ServerOption
	opts = append(opts, grpc.UnaryInterceptor(interceptor), grpc.StreamInterceptor(interceptorStream))
	//区块最多10M
	msgRecvOp := grpc.MaxMsgSize(11 * 1024 * 1024)     //设置最大接收数据大小位11M
	msgSendOp := grpc.MaxSendMsgSize(11 * 1024 * 1024) //设置最大发送数据大小为11M
	var keepparm keepalive.ServerParameters
	keepparm.Time = 5 * time.Minute
	keepparm.Timeout = 50 * time.Second
	keepparm.MaxConnectionIdle = 1 * time.Minute
	maxStreams := grpc.MaxConcurrentStreams(1000)
	keepOp := grpc.KeepaliveParams(keepparm)
	StatsOp := grpc.StatsHandler(&statshandler{})
	opts = append(opts, msgRecvOp, msgSendOp, keepOp, maxStreams, StatsOp)
	dl.server = grpc.NewServer(opts...)
	dl.p2pserver = pServer
	pb.RegisterP2PgserviceServer(dl.server, pServer)
	return dl
}

type statshandler struct{}

func (h *statshandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return context.WithValue(ctx, connCtxKey{}, info)
}

func (h *statshandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *statshandler) HandleConn(ctx context.Context, s stats.ConnStats) {
	if ctx == nil {
		return
	}
	tag, ok := getConnTagFromContext(ctx)
	if !ok {
		fmt.Println("can not get conn tag")
		return
	}

	ip, _, err := net.SplitHostPort(tag.RemoteAddr.String())
	if err != nil {
		return
	}
	connsMutex.Lock()
	defer connsMutex.Unlock()
	if _, ok := conns[ip]; !ok {
		conns[ip] = 0
	}
	switch s.(type) {
	case *stats.ConnBegin:
		conns[ip] = conns[ip] + 1
	case *stats.ConnEnd:
		conns[ip] = conns[ip] - 1
		if conns[ip] <= 0 {
			delete(conns, ip)
		}
		log.Debug("ip connend", "ip", ip, "n", conns[ip])
	default:
		log.Error("illegal ConnStats type\n")
	}
}

// HandleRPC 为空.
func (h *statshandler) HandleRPC(ctx context.Context, s stats.RPCStats) {}

type connCtxKey struct{}

var connsMutex sync.Mutex

var conns = make(map[string]uint)

func getConnTagFromContext(ctx context.Context) (*stats.ConnTagInfo, bool) {
	tag, ok := ctx.Value(connCtxKey{}).(*stats.ConnTagInfo)
	return tag, ok
}

func auth(checkIP string) bool {
	connsMutex.Lock()
	defer connsMutex.Unlock()
	count, ok := conns[checkIP]
	if ok && count > maxSamIPNum {
		log.Error("AuthCheck", "sameIP num:", count, "checkIP:", checkIP, "diffIP num:", len(conns))
		return false
	}

	return true
}
