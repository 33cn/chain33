package manage

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/stretchr/testify/assert"
	"io"
	"sort"
	"testing"
	"time"
)

type testConn struct {
	io.Closer
	network.ConnSecurity
	network.ConnMultiaddrs
	stat network.Stat
}

func (t testConn) ID() string {
	return ""
}

// NewStream constructs a new Stream over this conn.
func (t testConn) NewStream() (network.Stream, error) { return nil, nil }

// GetStreams returns all open streams over this conn.
func (t testConn) GetStreams() []network.Stream { return nil }

// Stat stores metadata pertaining to this conn.
func (t testConn) Stat() network.Stat {
	return t.stat
}

func newtestConn(stat network.Stat) network.Conn {
	return testConn{stat: stat}

}
func Test_SortConn(t *testing.T) {
	var testconn conns

	var s1, s2, s3 network.Stat
	s1.Opened = time.Now().Add(time.Second * 10)
	s2.Opened = time.Now().Add(time.Second * 15)
	s3.Opened = time.Now().Add(time.Minute)
	c1 := newtestConn(s1)
	c2 := newtestConn(s2)
	c3 := newtestConn(s3)

	testconn = append(testconn, c1, c2, c3)
	sort.Sort(testconn)
	assert.Equal(t, testconn[0], c3)
	assert.Equal(t, testconn[2], c1)
}
