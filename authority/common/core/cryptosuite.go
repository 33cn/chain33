/*
Copyright IBM Corp. 2016, SecureKey Technologies Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/*
Notice: This file is a modified version of ‘internal/github.com/hyperledger/fabric/bccsp/bccsp.go’
where interfaces and functions are removed to minimize for Hyperledger Fabric SDK Go usage.

CryptoSuite interface defined in this file acts as a wrapper for
‘github.com.hyperledger.fabric.bccsp.BCCSP’ interface
*/

package core

import (
	"hash"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
)

//CryptoSuite adaptor for all bccsp functionalities used by SDK
type CryptoSuite interface {

	// KeyImport imports a key from its raw representation using opts.
	// The opts argument should be appropriate for the primitive used.
	KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error)

	// Hash hashes messages msg using options opts.
	// If opts is nil, the default hash function will be used.
	Hash(msg []byte, opts bccsp.HashOpts) (hash []byte, err error)

	// GetHash returns and instance of hash.Hash using options opts.
	// If opts is nil, the default hash function will be returned.
	GetHash(opts bccsp.HashOpts) (h hash.Hash, err error)

	// Sign signs digest using key k.
	// The opts argument should be appropriate for the algorithm used.
	//
	// Note that when a signature of a hash of a larger message is needed,
	// the caller is responsible for hashing the larger message and passing
	// the hash (as digest).
	Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error)
}

//CryptoSuiteConfig contains sdk configuration items for cryptosuite.
type CryptoSuiteConfig interface {
	SecurityAlgorithm() string
	SecurityLevel() int
}
