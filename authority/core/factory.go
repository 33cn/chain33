/*
Copyright IBM Corp. 2017 All Rights Reserved.

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