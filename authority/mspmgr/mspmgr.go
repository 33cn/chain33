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
	"sync"

	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
)

// LoadLocalMsp loads the local MSP from the specified directory
func LoadLocalMsp(dir string, cryptoConf *cryptosuite.CryptoConfig) error {
	conf, err := GetLocalMspConfig(dir, cryptoConf)
	if err != nil {
		return err
	}

	return GetLocalMSP().Setup(conf)
}

// FIXME: AS SOON AS THE CHAIN MANAGEMENT CODE IS COMPLETE,
// THESE MAPS AND HELPSER FUNCTIONS SHOULD DISAPPEAR BECAUSE
// OWNERSHIP OF PER-CHAIN MSP MANAGERS WILL BE HANDLED BY IT;
// HOWEVER IN THE INTERIM, THESE HELPER FUNCTIONS ARE REQUIRED

var m sync.Mutex
var localMsp MSP

// GetLocalMSP returns the local msp (and creates it if it doesn't exist)
func GetLocalMSP() MSP {
	var lclMsp MSP
	var created bool = false
	{
		m.Lock()
		defer m.Unlock()

		lclMsp = localMsp
		if lclMsp == nil {
			var err error
			created = true
			lclMsp, err = NewBccspMsp()
			if err != nil {
				mspLogger.Crit("Failed to initialize local MSP, received err %s", err)
			}
			localMsp = lclMsp
		}
	}

	if created {
		mspLogger.Debug("Created new local MSP")
	} else {
		mspLogger.Debug("Returning existing local MSP")
	}

	return lclMsp
}
