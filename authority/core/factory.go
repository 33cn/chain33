package core

import (
	"errors"
)

func GetLocalValidator(dir string) (Validator, error) {
	conf, err := GetAuthConfig(dir)
	if err != nil {
		return nil, err
	}

	lclValidator, err := NewEcdsaValidator()
	if err != nil {
		authLogger.Error("Failed to initialize local MSP", "Error", err)
		return nil, errors.New("Failed to initialize local MSP")
	}

	err = lclValidator.Setup(conf)
	if err != nil {
		authLogger.Error("Failed to set up local MSP config", "Error", err)
		return nil, errors.New("Failed to initialize local MSP")
	}

	return lclValidator, nil
}
