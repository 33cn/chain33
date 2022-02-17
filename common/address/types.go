// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address

// Config address driver config
// [address]
// defaultDriver="btc"
// [address.enableHeight]
// btc=0
// btcMultiSign=0
// eth=-1
type Config struct {

	// DefaultDriver config default driver
	DefaultDriver string `json:"defaultDriver,omitempty"`
	// EnableHeight enable driver at specific block height
	EnableHeight map[string]int64 `json:"enableHeight,omitempty"`
}
