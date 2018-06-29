package upnp

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/inconshreveable/log15"
	cmn "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/common"
)

type UPNPCapabilities struct {
	PortMapping bool
	Hairpin     bool
}

var (
	probelog = log15.New("module", "tendermint-probe")
)

func makeUPNPListener(intPort int, extPort int) (NAT, net.Listener, net.IP, error) {
	nat, err := Discover()
	if err != nil {
		return nil, nil, nil, errors.Errorf("NAT upnp could not be discovered: %v", err)
	}
	probelog.Info(cmn.Fmt("ourIP: %v", nat.(*upnpNAT).ourIP))

	ext, err := nat.GetExternalAddress()
	if err != nil {
		return nat, nil, nil, errors.Errorf("External address error: %v", err)
	}
	probelog.Info(cmn.Fmt("External address: %v", ext))

	port, err := nat.AddPortMapping("tcp", extPort, intPort, "Tendermint UPnP Probe", 0)
	if err != nil {
		return nat, nil, ext, errors.Errorf("Port mapping error: %v", err)
	}
	probelog.Info(cmn.Fmt("Port mapping mapped: %v", port))

	// also run the listener, open for all remote addresses.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", intPort))
	if err != nil {
		return nat, nil, ext, errors.Errorf("Error establishing listener: %v", err)
	}
	return nat, listener, ext, nil
}

func testHairpin(listener net.Listener, extAddr string) (supportsHairpin bool) {
	// Listener
	go func() {
		inConn, err := listener.Accept()
		if err != nil {
			probelog.Info(cmn.Fmt("Listener.Accept() error: %v", err))
			return
		}
		probelog.Info(cmn.Fmt("Accepted incoming connection: %v -> %v", inConn.LocalAddr(), inConn.RemoteAddr()))
		buf := make([]byte, 1024)
		n, err := inConn.Read(buf)
		if err != nil {
			probelog.Info(cmn.Fmt("Incoming connection read error: %v", err))
			return
		}
		probelog.Info(cmn.Fmt("Incoming connection read %v bytes: %X", n, buf))
		if string(buf) == "test data" {
			supportsHairpin = true
			return
		}
	}()

	// Establish outgoing
	outConn, err := net.Dial("tcp", extAddr)
	if err != nil {
		probelog.Info(cmn.Fmt("Outgoing connection dial error: %v", err))
		return
	}

	n, err := outConn.Write([]byte("test data"))
	if err != nil {
		probelog.Info(cmn.Fmt("Outgoing connection write error: %v", err))
		return
	}
	probelog.Info(cmn.Fmt("Outgoing connection wrote %v bytes", n))

	// Wait for data receipt
	time.Sleep(1 * time.Second)
	return
}

func Probe() (caps UPNPCapabilities, err error) {
	probelog.Info("Probing for UPnP!")

	intPort, extPort := 8001, 8001

	nat, listener, ext, err := makeUPNPListener(intPort, extPort)
	if err != nil {
		return
	}
	caps.PortMapping = true

	// Deferred cleanup
	defer func() {
		if err := nat.DeletePortMapping("tcp", intPort, extPort); err != nil {
			probelog.Error(cmn.Fmt("Port mapping delete error: %v", err))
		}
		if err := listener.Close(); err != nil {
			probelog.Error(cmn.Fmt("Listener closing error: %v", err))
		}
	}()

	supportsHairpin := testHairpin(listener, fmt.Sprintf("%v:%v", ext, extPort))
	if supportsHairpin {
		caps.Hairpin = true
	}

	return
}
