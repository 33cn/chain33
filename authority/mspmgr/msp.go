package mspmgr

type MSP interface {

	// IdentityDeserializer interface needs to be implemented by MSP
	IdentityDeserializer

	// Setup the MSP instance according to configuration information
	Setup(config *MSPConfig) error

	// GetIdentifier returns the provider identifier
	GetIdentifier() (string, error)

	// Validate checks whether the supplied identity is valid
	Validate(id MSPIdentity) error
}

// IdentityDeserializer is implemented by both MSPManger and MSP
type IdentityDeserializer interface {
	// DeserializeIdentity deserializes an identity.
	// Deserialization will fail if the identity is associated to
	// an msp that is different from this one that is performing
	// the deserialization.
	DeserializeIdentity(serializedIdentity []byte) (MSPIdentity, error)
}

type MSPConfig struct {
	RootCerts [][]byte
	IntermediateCerts [][]byte
	Admins [][]byte
	RevocationList [][]byte
	CryptoConfig *MSPCryptoConfig
}

type MSPCryptoConfig struct {
	SignatureHashFamily string
	IdentityIdentifierHashFunction string
}

// Identity interface defining operations associated to a "certificate".
// That is, the public part of the identity could be thought to be a certificate,
// and offers solely signature verification capabilities. This is to be used
// at the peer side when verifying certificates that transactions are signed
// with, and verifying signatures that correspond to these certificates.///
type MSPIdentity interface {
	// Validate uses the rules that govern this identity to validate it.
	// E.g., if it is a fabric TCert implemented as identity, validate
	// will check the TCert signature against the assumed root certificate
	// authority.
	Validate() error

	// Verify a signature over some message using this identity as reference
	Verify(msg []byte, sig []byte) error
}