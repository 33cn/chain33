// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "crypto/rand"
import "fmt"
import btcutil "github.com/33cn/chain33/wallet/bipwallet/btcutilecc"

func main() {
	// generate keys for Alice and Bob
	alice, err := btcutil.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Println("Generate alice Key err", err)
	}
	bob, err := btcutil.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Println("Generate bob Key err", err)
	}
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
