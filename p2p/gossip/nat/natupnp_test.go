// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package nat

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/huin/goupnp/httpu"
)

// fakeIGD presents itself as a discoverable UPnP device which sends
// canned responses to HTTPU and HTTP requests.
type fakeIGD struct {
	t *testing.T // for logging

	listener      net.Listener
	mcastListener *net.UDPConn

	// This should be a complete HTTP response (including headers).
	// It is sent as the response to any sspd packet. Any occurrence
	// of "{{listenAddr}}" is replaced with the actual TCP listen
	// address of the HTTP server.
	ssdpResp string
	// This one should contain XML payloads for all requests
	// performed. The keys contain method and path, e.g. "GET /foo/bar".
	// As with ssdpResp, "{{listenAddr}}" is replaced with the TCP
	// listen address.
	httpResps map[string]string
}

// httpu.Handler
func (dev *fakeIGD) ServeMessage(r *http.Request) {
	dev.t.Logf(`HTTPU request %s %s`, r.Method, r.RequestURI)
	conn, err := net.Dial("udp4", r.RemoteAddr)
	if err != nil {
		fmt.Printf("reply Dial error: %v", err)
		return
	}
	defer conn.Close()
	io.WriteString(conn, dev.replaceListenAddr(dev.ssdpResp))
}

// http.Handler
func (dev *fakeIGD) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if resp, ok := dev.httpResps[r.Method+" "+r.RequestURI]; ok {
		dev.t.Logf(`HTTP request "%s %s" --> %d`, r.Method, r.RequestURI, 200)
		io.WriteString(w, dev.replaceListenAddr(resp))
	} else {
		dev.t.Logf(`HTTP request "%s %s" --> %d`, r.Method, r.RequestURI, 404)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (dev *fakeIGD) replaceListenAddr(resp string) string {
	return strings.Replace(resp, "{{listenAddr}}", dev.listener.Addr().String(), -1)
}

func (dev *fakeIGD) listen() (err error) {
	if dev.listener, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
		return err
	}
	laddr := &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900}
	if dev.mcastListener, err = net.ListenMulticastUDP("udp", nil, laddr); err != nil {
		dev.listener.Close()
		return err
	}
	return nil
}

func (dev *fakeIGD) serve() {
	go httpu.Serve(dev.mcastListener, dev)
	go http.Serve(dev.listener, dev)
}

func (dev *fakeIGD) close() {
	dev.mcastListener.Close()
	dev.listener.Close()
}
