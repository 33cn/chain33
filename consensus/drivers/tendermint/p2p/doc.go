// NCNL is No Copyright and No License

package p2p

/*
# 本包实现dpos的p2p网络协议

## 概念

- SeedServer: 网络默认配置参数, 可更新: 表示种子服务器, 网络所有的节点Node连接此服务器, 注册IP和端口; 每个Node通过SeedServer获取其他NodeList
- Node: 表示p2p协议网络中的一个节点
- Client: 非p2p网络节点, 和某个Node连接, 使用rpc或者websocket通信
- Peer: 表示远程对端, 用于向对端发送和接收数据
- discover: 在SeedServer不可用的情况下, 发现NodeList. 使用DHT技术
- NodeDB: 存储网络配置参数和NodeList列表
- Server: 监听p2p协议端口, 管理PeerList, 和ServiceList
- Dailer: 用于呼叫Node, 创建Peer
- handshake: 和Peer握手过程, 之后进行通信
- Msg: 用于封装p2p消息, 使用protobuf封装
- Service: 节点提供的某种服务, 向Server注册, 由它处理Peer的消息

##过程

```go

func (s *Server) Run() error {
	seed, err := Dail(s.conf.seedServer)
	if err != nil {
		log.Error(err)
		go discover()
	} else {
		s.nodeList = seed.GetNodeList()
	}
	list := s.db.GetNodeList()
	for k, v := range list {
		_, ok := s.nodeList[k]
		if !ok {
			s.nodeList[k] = v
		}
	}
	for k, v := range s.nodeList {
		conn := Dail(v)
		peer := NewPeer(conn, s)
		if peer.Handshake() {
			s.peerList[k] = peer
		} else {
			peer.Close()
		}
		go peer.Serve()
	}
	l, err := net.Listen("tcp", peerService)
	if err != nil {
		glog.Error(err)
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Error(err)
			return err
		}
		peer := NewPeer(conn, s)
		if peer.Handshake() {
			s.peerList[k] = peer
		} else {
			peer.Close()
		}
		go peer.Serve()
	}
	return nil
}

```


*/
