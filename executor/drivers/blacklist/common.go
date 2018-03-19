package blacklist

import (
	mrand "math/rand"
	"time"
	"fmt"
)
func generateAddr()string{
	rand := mrand.New(mrand.NewSource(time.Now().UnixNano()))
    addr :=fmt.Sprintf("%06v", rand.Uint64())
    fmt.Println(addr)
    return addr
}
func generateTxId()string{
	rand := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	addr :=fmt.Sprintf("%06v", rand.Int31n(1000000))
	fmt.Println(addr)
	return addr
}