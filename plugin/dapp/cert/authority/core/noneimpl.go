package core

type noneValidator struct {
}

func NewNoneValidator() (Validator, error) {
	return &noneValidator{}, nil
}

func (validator *noneValidator) Setup(conf *AuthConfig) error {
	return nil
}

func (validator *noneValidator) Validate(certByte []byte, pubKey []byte) error {
	return nil
}

func (Validator *noneValidator) GetCertFromSignature(signature []byte) ([]byte, error) {
	return []byte(""), nil
}
