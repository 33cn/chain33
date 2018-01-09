package p2p

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/common/crypto"
	pb "code.aliyun.com/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type MConnection struct {
	nodeInfo      **NodeInfo
	gconn         *grpc.ClientConn
	conn          pb.P2PgserviceClient // source connection
	config        *MConnConfig
	key           string //privkey
	quit          chan bool
	versionDone   chan struct{}
	remoteAddress *NetAddress //peer 的地址
	//pingTimer     *RepeatTimer // send pings periodically
	//versionTimer *RepeatTimer
	peer        *peer
	sendMonitor *Monitor
}

// MConnConfig is a MConnection configuration.
type MConnConfig struct {
	gconn *grpc.ClientConn
	conn  pb.P2PgserviceClient
}

// DefaultMConnConfig returns the default config.
func DefaultMConnConfig() *MConnConfig {
	return &MConnConfig{}
}

func NewTemMConnConfig(gconn *grpc.ClientConn, conn pb.P2PgserviceClient) *MConnConfig {
	return &MConnConfig{
		gconn: gconn,
		conn:  conn,
	}
}

// NewMConnection wraps net.Conn and creates multiplex connection
func NewMConnection(conn *grpc.ClientConn, remote *NetAddress, peer *peer) *MConnection {

	mconn := &MConnection{
		gconn: conn,
		conn:  pb.NewP2PgserviceClient(conn),
		//pingTimer:   NewRepeatTimer("ping", PingTimeout),
		sendMonitor: NewMonitor(),
		peer:        peer,
		quit:        make(chan bool),
	}
	mconn.nodeInfo = peer.nodeInfo
	mconn.remoteAddress = remote

	return mconn

}

func NewMConnectionWithConfig(cfg *MConnConfig) *MConnection {
	mconn := &MConnection{
		gconn: cfg.gconn,
		conn:  cfg.conn,
	}

	return mconn
}

func (c *MConnection) signature(in *pb.P2PPing) (*pb.P2PPing, error) {
	data := pb.Encode(in)

	cr, err := crypto.New(pb.GetSignatureTypeName(pb.SECP256K1))
	if err != nil {
		log.Error("CryPto Error", "Error", err.Error())
		return nil, err
	}
	pribyts, err := hex.DecodeString(c.key)
	if err != nil {
		log.Error("DecodeString Error", "Error", err.Error())
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(pribyts)
	if err != nil {
		log.Error("Load PrivKey", "Error", err.Error())
		return nil, err
	}
	in.Sign = new(pb.Signature)
	in.Sign.Signature = priv.Sign(data).Bytes()
	in.Sign.Ty = pb.SECP256K1
	in.Sign.Pubkey = priv.PubKey().Bytes()
	return in, nil
}

// sendRoutine polls for packets to send from channels.
func (c *MConnection) pingRoutine() {

	var pingtimes int64
	ticker := time.NewTicker(PingTimeout)
	defer ticker.Stop()
FOR_LOOP:
	for {

		select {
		case <-ticker.C:
			randNonce := rand.Int31n(102040)
			in, err := c.signature(&pb.P2PPing{Nonce: int64(randNonce), Addr: ExternalAddr, Port: int32((*c.nodeInfo).externalAddr.Port)})
			if err != nil {
				log.Error("Signature", "Error", err.Error())
				continue
			}
			log.Debug("SEND PING", "Peer", c.remoteAddress.String(), "nonce", randNonce)
			r, err := c.conn.Ping(context.Background(), in)
			if err != nil {
				c.sendMonitor.Update(false)
				if pingtimes == 0 {
					(*c.nodeInfo).monitorChan <- c.peer
				}
				continue
			}

			log.Debug("RECV PONG", "resp:", r.Nonce, "Ping nonce:", randNonce)
			c.sendMonitor.Update(true)
			pingtimes++

		case <-c.quit:
			break FOR_LOOP

		}

	}

}

func (c *MConnection) sendVersion() error {
	client := (*c.nodeInfo).q.GetClient()
	msg := client.NewMessage("blockchain", pb.EventGetBlockHeight, nil)
	client.Send(msg, true)
	rsp, err := client.Wait(msg)
	if err != nil {
		log.Error("GetHeight", "Error", err.Error())
		return err
	}

	blockheight := rsp.GetData().(*pb.ReplyBlockHeight).GetHeight()
	randNonce := rand.Int31n(102040)
	in, err := c.signature(&pb.P2PPing{Nonce: int64(randNonce), Addr: ExternalAddr, Port: int32((*c.nodeInfo).externalAddr.Port)})
	if err != nil {
		log.Error("Signature", "Error", err.Error())
		return err
	}
	addrfrom := fmt.Sprintf("%v:%v", ExternalAddr, (*c.nodeInfo).externalAddr.Port)
	(*c.nodeInfo).blacklist.Add(addrfrom)
	resp, err := c.conn.Version2(context.Background(), &pb.P2PVersion{Version: (*c.nodeInfo).cfg.GetVersion(), Service: SERVICE, Timestamp: time.Now().Unix(),
		AddrRecv: c.remoteAddress.String(), AddrFrom: addrfrom, Nonce: int64(rand.Int31n(102040)),
		UserAgent: hex.EncodeToString(in.Sign.GetPubkey()), StartHeight: blockheight})
	if err != nil {
		c.peer.version.Set(false)
		if strings.EqualFold(err.Error(), VersionNotSupport) == true {
			(*c.nodeInfo).monitorChan <- c.peer
		}

		return err
	}

	log.Debug("SHOW VERSION BACK", "VersionBack", resp)
	return nil
}

func (c *MConnection) getAddr() ([]string, error) {
	resp, err := c.conn.GetAddr(context.Background(), &pb.P2PGetAddr{Nonce: int64(rand.Int31n(102040))})
	if err != nil {

		c.sendMonitor.Update(false)
		return nil, err
	}

	log.Debug("GetAddr Resp", "Resp", resp, "addrlist", resp.Addrlist)
	c.sendMonitor.Update(true)
	return resp.Addrlist, nil
}

// OnStart implements BaseService
func (c *MConnection) start() error { //启动Mconnection，每一个MConnection 会在启动的时候启动SendRoutine,RecvRoutine

	go c.pingRoutine() //创建发送Routine
	return nil
}

func (c *MConnection) close() {
	c.gconn.Close()
}

func (c *MConnection) stop() {

	c.sendMonitor.Stop()
	c.gconn.Close()
	c.quit <- false
	log.Debug("Mconnection", "Close", "^_^!")
}
