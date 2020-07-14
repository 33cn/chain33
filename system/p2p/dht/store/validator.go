package store

type validator struct {
}

//Validate for testing
func (validator) Validate(_ string, _ []byte) error {
	return nil
}

//Select for testing
func (validator) Select(_ string, _ [][]byte) (int, error) {
	return 0, nil
}
