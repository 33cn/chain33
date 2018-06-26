package core

type Validator interface {
	// Setup the MSP instance according to configuration information
	Setup(config *AuthConfig) error

	// Validate checks whether the supplied identity is valid
	Validate(cert []byte) error
}

type AuthConfig struct {
	RootCerts         [][]byte
	IntermediateCerts [][]byte
	RevocationList    [][]byte
}
