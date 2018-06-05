/*
Copyright IBM Corp. 2016 All Rights Reserved.

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
Notice: This file has been modified for Hyperledger Fabric SDK Go usage.
Please review third_party pinning scripts and patches for more details.
*/

package bccsp

const (
	// ECDSA Elliptic Curve Digital Signature Algorithm (key gen, import, sign, verify),
	// at default security level.
	// Each BCCSP may or may not support default security level. If not supported than
	// an error will be returned.
	ECDSA = "ECDSA"

	// ECDSA Elliptic Curve Digital Signature Algorithm over P-256 curve
	ECDSAP256 = "ECDSAP256"
	// RSA at the default security level.
	// Each BCCSP may or may not support default security level. If not supported than
	// an error will be returned.
	RSA = "RSA"

	// AES Advanced Encryption Standard at the default security level.
	// Each BCCSP may or may not support default security level. If not supported than
	// an error will be returned.
	AES = "AES"

	// HMAC keyed-hash message authentication code
	HMAC = "HMAC"

	// SHA Secure Hash Algorithm using default family.
	// Each BCCSP may or may not support default security level. If not supported than
	// an error will be returned.
	SHA = "SHA"

	// SHA2 is an identifier for SHA2 hash family
	SHA2 = "SHA2"
	// SHA3 is an identifier for SHA3 hash family
	SHA3 = "SHA3"

	// SHA256
	SHA256 = "SHA256"
	// SHA384
	SHA384 = "SHA384"
	// SHA3_256
	SHA3_256 = "SHA3_256"
	// SHA3_384
	SHA3_384 = "SHA3_384"

	// X509Certificate Label for X509 certificate related operation
	X509Certificate = "X509Certificate"
)

// ECDSAPKIXPublicKeyImportOpts contains options for ECDSA public key importation in PKIX format
type ECDSAPKIXPublicKeyImportOpts struct {
	Temporary bool
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *ECDSAPKIXPublicKeyImportOpts) Algorithm() string {
	return ECDSA
}

// ECDSAPrivateKeyImportOpts contains options for ECDSA secret key importation in DER format
// or PKCS#8 format.
type ECDSAPrivateKeyImportOpts struct {
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *ECDSAPrivateKeyImportOpts) Algorithm() string {
	return ECDSA
}

// ECDSAGoPublicKeyImportOpts contains options for ECDSA key importation from ecdsa.PublicKey
type ECDSAGoPublicKeyImportOpts struct {
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *ECDSAGoPublicKeyImportOpts) Algorithm() string {
	return ECDSA
}

// AES256ImportKeyOpts contains options for importing AES 256 keys.
type AES256ImportKeyOpts struct {
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *AES256ImportKeyOpts) Algorithm() string {
	return AES
}

// HMACImportKeyOpts contains options for importing HMAC keys.
type HMACImportKeyOpts struct {
	Temporary bool
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *HMACImportKeyOpts) Algorithm() string {
	return HMAC
}

// SHAOpts contains options for computing SHA.
type SHAOpts struct {
}

// Algorithm returns the hash algorithm identifier (to be used).
func (opts *SHAOpts) Algorithm() string {
	return SHA
}

// ECDSAGoPublicKeyImportOpts contains options for RSA key importation from rsa.PublicKey
type RSAGoPublicKeyImportOpts struct {
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *RSAGoPublicKeyImportOpts) Algorithm() string {
	return RSA
}

// X509PublicKeyImportOpts contains options for importing public keys from an x509 certificate
type X509PublicKeyImportOpts struct {
}

// Algorithm returns the key importation algorithm identifier (to be used).
func (opts *X509PublicKeyImportOpts) Algorithm() string {
	return X509Certificate
}
