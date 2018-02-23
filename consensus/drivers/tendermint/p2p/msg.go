// NCNL is No Copyright and No License

package p2p

// "code.aliyun.com/chain33/chain33/consensus/drivers/pos33/crypto"

const (
	CloseSvc     = "close"
	PeerErrorSvc = "peerError"
)

type Msg struct {
	Service string
	Data    []byte
}

type IPeer interface {
	WriteMsg(m *Msg) error
	RegService(name string, svc Service)
	RemoteAddr() string
	Who(string) string
	Close()
}

type NodeMsg struct {
	IPeer
	*Msg
}
