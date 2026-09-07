// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import "errors"

var (
	errNilPoint      = errors.New("cggmp: nil ec point")
	errPPKTimeout    = errors.New("cggmp: partial public key exchange timed out")
	errMissingSelfBk = errors.New("cggmp: missing self bk in dkg result")
	errNilResult     = errors.New("cggmp: nil result")
)
