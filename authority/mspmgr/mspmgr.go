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

package mspmgr

import (
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"errors"
)

var localMsp MSP

// LoadLocalMsp loads the local MSP from the specified directory
func LoadLocalMsp(dir string, cryptoConf *cryptosuite.CryptoConfig) error {
	conf, err := GetLocalMspConfig(dir, cryptoConf)
	if err != nil {
		return err
	}

	lclMsp, err := NewBccspMsp()
	if err != nil {
		mspLogger.Error("Failed to initialize local MSP", "Error", err)
		return errors.New("Failed to initialize local MSP")
	}
	localMsp = lclMsp
	return localMsp.Setup(conf)
}

func GetLocalMSP() MSP {
	return localMsp
}
