package p2pnext

import (
	"net"
	"strings"
)

type Monitor struct {
}

func (m *Monitor) MonitorPstore() {
	//
}

func getNodeLocalAddr() string {
	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	defer conn.Close()
	log.Info("CheckNodeAddr", "Addr:", strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}
