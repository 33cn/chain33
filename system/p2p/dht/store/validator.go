package store

type Validator struct {

}

func (Validator) Validate(_ string, _ []byte) error {
	return nil
}

func (Validator) Select(_ string, _ [][]byte) (int, error) {
	return 0, nil
}
