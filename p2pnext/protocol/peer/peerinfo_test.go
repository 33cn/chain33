package peer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testSetExternalAddr(t *testing.T) {
	var peerinfoProto = new(PeerInfoProtol)

	peerinfoProto.SetExternalAddr("192.168.1.1")
	assert.Empty(t, peerinfoProto.GetExternalAddr())
	peerinfoProto.SetExternalAddr("/ip4/192.168.1.1/13802")
	assert.Equal(t, "192.168.1.1", peerinfoProto.GetExternalAddr())

}

func TestPeerInfoFunc(t *testing.T) {
	testSetExternalAddr(t)
}
