package core

import (
	"errors"
)

func GetLocalValidator(authConfig *AuthConfig, signType int) (Validator, error) {
	lclValidator, err := NewEcdsaValidator()
	if err != nil {
		authLogger.Error("Failed to initialize local MSP", "Error", err)
		return nil, errors.New("Failed to initialize local MSP")
	}

	err = lclValidator.Setup(authConfig)
	if err != nil {
		authLogger.Error("Failed to set up local MSP config", "Error", err)
		return nil, errors.New("Failed to initialize local MSP")
	}

	return lclValidator, nil
}
