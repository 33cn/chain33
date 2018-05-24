package common

import (
	"testing"
	"fmt"
)

func TestAddressBig(t *testing.T) {
	saddr := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	addr := StringToAddress(saddr)
	baddr := addr.Big()
	naddr := BigToAddress(baddr)
	if saddr != naddr.NormalString() {
		t.Fail()
	}
}


func TestAddressBytes(t *testing.T) {
	addr := BytesToAddress([]byte{1})

	fmt.Println(addr.Str())
}