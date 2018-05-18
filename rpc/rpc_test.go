package rpc

import (
	"testing"
)

func TestDecodeUserWrite(t *testing.T) {
	payload := []byte("#md#hello#world")
	data := decodeUserWrite(payload)
	if data.Topic != "md" {
		t.Error("topic expect md, current", data.Topic)
		return
	}
	if data.Content != "hello#world" {
		t.Error("topic expect hello#world, current", data.Content)
		return
	}

	payload = []byte("hello#world")
	data = decodeUserWrite(payload)
	if data.Topic != "" {
		t.Error("topic expect empty, current", data.Topic)
		return
	}
	if data.Content != "hello#world" {
		t.Error("topic expect hello#world, current", data.Content)
		return
	}
}
