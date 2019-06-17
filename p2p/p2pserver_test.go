package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AddDelStream(t *testing.T) {

	s := NewP2pServer()
	peerName := "testpeer"
	delChan := s.addStreamHandler(peerName)
	//replace
	dataChan := s.addStreamHandler(peerName)

	_, ok := <-delChan
	assert.False(t, ok)

	//del old
	s.deleteStream(peerName, delChan)
	_, ok = s.streams[peerName]
	assert.True(t, ok)
	//del nil
	s.deleteStream("", delChan)
	//del exist
	s.deleteStream(peerName, dataChan)

	_, ok = s.streams[peerName]
	assert.False(t, ok)
}
