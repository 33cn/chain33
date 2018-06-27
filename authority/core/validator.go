package core

type Validator interface {
	Setup(config *AuthConfig) error

	Validate(cert []byte) error
}

type AuthConfig struct {
	RootCerts         [][]byte
	IntermediateCerts [][]byte
	RevocationList    [][]byte
}
