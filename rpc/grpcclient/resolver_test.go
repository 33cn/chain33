// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/resolver"
)

func TestParseTarget(t *testing.T) {
	_, _, err := parseTarget("", "8802")
	assert.NotNil(t, err)

	// IP address
	host, port, err := parseTarget("127.0.0.1", "8802")
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1", host)
	assert.Equal(t, "8802", port)

	// IPv6 address
	host, port, err = parseTarget("::1", "8802")
	assert.Nil(t, err)
	assert.Equal(t, "::1", host)
	assert.Equal(t, "8802", port)

	// Host:Port
	host, port, err = parseTarget("localhost:8802", "8802")
	assert.Nil(t, err)
	assert.Equal(t, "localhost", host)
	assert.Equal(t, "8802", port)

	// Missing port with colon
	_, _, err = parseTarget("localhost:", "8802")
	assert.NotNil(t, err)

	// Bare host (uses default port)
	host, port, err = parseTarget("example.com", "8802")
	assert.Nil(t, err)
	assert.Equal(t, "example.com", host)
	assert.Equal(t, "8802", port)
}

func TestNewMultipleURL(t *testing.T) {
	url := NewMultipleURL("localhost:8802,localhost:8803")
	assert.Equal(t, "multiple:///localhost:8802,localhost:8803", url)
}

func TestMultipleResolverResolveNowAndClose(t *testing.T) {
	r := &multipleResolver{}
	r.Close()
	assert.NotPanics(t, func() {
		r.ResolveNow(resolver.ResolveNowOptions{})
	})
}

func TestMultipleBuilderScheme(t *testing.T) {
	b := &multipleBuilder{}
	assert.Equal(t, "multiple", b.Scheme())
}
