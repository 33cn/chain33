package p2pnext

import (
	"net"
	"strings"
)

type Monitor struct {
}

func getNodeLocalAddr() string {
	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	defer conn.Close()
	logger.Info("CheckNodeAddr", "Addr:", strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}
