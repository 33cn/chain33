package core

import (
	"errors"

	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
)

func GetLocalValidator(authConfig *AuthConfig, signType int) (Validator, error) {
	var lclValidator Validator
	var err error

	if signType == ty.AUTH_ECDSA {
		lclValidator = NewEcdsaValidator()
	} else if signType == ty.AUTH_SM2 {
		lclValidator = NewGmValidator()
	} else {
		return nil, ty.ErrUnknowAuthSignType
	}

	err = lclValidator.Setup(authConfig)
	if err != nil {
		authLogger.Error("Failed to set up local validator config", "Error", err)
		return nil, errors.New("Failed to initialize local validator")
	}

	return lclValidator, nil
}
