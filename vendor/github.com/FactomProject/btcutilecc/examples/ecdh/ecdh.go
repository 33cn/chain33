package main

import "crypto/rand"
import "fmt"
import "github.com/mndrix/btcutil"

func main() {
	// generate keys for Alice and Bob
	alice, _ := btcutil.GenerateKey(rand.Reader)
	bob, _ := btcutil.GenerateKey(rand.Reader)
	fmt.Printf("Alice:\t%x\n\t%x\n", alice.D, alice.PublicKey.X)
	fmt.Printf("Bob:\t%x\n\t%x\n", bob.D, bob.PublicKey.X)
	fmt.Println("")

	// Alice calculates shared secret
	aliceShared := btcutil.ECDH(alice, &bob.PublicKey)
	fmt.Printf("Alice: %x\n", aliceShared)

	// Bob calculates shared secret
	bobShared := btcutil.ECDH(bob, &alice.PublicKey)
	fmt.Printf("Bob:   %x\n", bobShared)
}
