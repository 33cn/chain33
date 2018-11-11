// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "crypto/rand"
import "fmt"
import btcutil "github.com/33cn/chain33/wallet/bipwallet/btcutilecc"

func main() {
	signer := new(btcutil.BlindSignerState)
	requester := new(btcutil.BlindRequesterState)

	// requester: message that needs to be blind signed
	m, err := btcutil.RandFieldElement(rand.Reader)
	maybePanic(err)
	fmt.Printf("m = %x\n", m)

	// requester: ask signer to start the protocol
	Q, R := btcutil.BlindSession(signer)
	fmt.Println("")

	// requester: blind message
	mHat := btcutil.BlindMessage(requester, Q, R, m)

	// signer: create blind signature
	sHat := btcutil.BlindSign(signer, R, mHat)

	// requester extracts real signature
	sig := btcutil.BlindExtract(requester, sHat)
	sig.M = m
	fmt.Printf("sig =\t%x\n\t%x\n", sig.S, sig.F.X)

	// onlooker verifies signature
	if btcutil.BlindVerify(Q, sig) {
		fmt.Printf("valid signature\n")
	}
}

func maybePanic(err error) {
	if err != nil {
		panic(err)
	}
}
