// NCNL is No Copyright and No License

package p2p

import (
	"encoding/gob"
	"log"
	"net"
	// "code.aliyun.com/chain33/chain33/consensus/drivers/pos33/crypto"
)

type Service interface {
	ServeMsg(IPeer, *Msg)
}

type Peer struct {
	who     string
	c       net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	padding chan *Msg
	packing bool
	svcMap  map[string][]Service
}

func NewPeer(c net.Conn) *Peer {
	p := &Peer{
		c:       c,
		enc:     gob.NewEncoder(c),
		dec:     gob.NewDecoder(c),
		padding: make(chan *Msg, 16),
		svcMap:  make(map[string][]Service),
	}
	return p
}

func (p *Peer) RegService(name string, svc Service) {
	p.svcMap[name] = append(p.svcMap[name], svc)
}

func (p *Peer) HandleMsg(w IPeer, m *Msg) {
	ss, ok := p.svcMap[m.Service]
	if !ok {
		log.Println("Peer::HandleMsg error: ", m.Service, " is NOT regester.", p.who, p.RemoteAddr())
		return
	}
	for _, x := range ss {
		x.ServeMsg(w, m)
	}
}

func (p *Peer) WriteMsg(m *Msg) error {
	p.padding <- m

	return nil
}

func (p *Peer) Serve() {
	go func() {
		for m := range p.padding {
			err := p.enc.Encode(m)
			if err != nil {
				log.Println("peer send error:  ", err)
				p.Close()
				break
			}
		}
	}()

	for {
		m := &Msg{}
		err := p.dec.Decode(m)
		if err != nil {
			log.Println("peer recv error: ", err)
			m = &Msg{Service: CloseSvc}
			p.HandleMsg(p, m)
			p.Close()
			return
		}
		p.HandleMsg(p, m)
	}
}

func (p *Peer) Close() {
	p.c.Close()
}

func (p *Peer) RemoteAddr() string {
	return p.c.RemoteAddr().String()
}

func (p *Peer) Who(who string) string {
	if who == "" {
		return p.RemoteAddr()
	}
	p.who = who
	return who
}
