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
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"

	"io/ioutil"

	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/ca"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/msp"
)

const (
	commonName = "ca"
)

type Config struct {
	Name    []string
	OrgName string
}

//command line flags
var (
	app = kingpin.New("cryptogen", "Utility for generating key material")

	gen        = app.Command("generate", "Generate key material")
	outputDir  = gen.Flag("output", "The output directory in which to place artifacts").Default("../../../cmd/chain33/authdir/crypto").String()
	configFile = gen.Flag("config", "The configuration file to use").Default("./new.json").File()
)

func main() {
	kingpin.Version("0.0.1")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	// "generate" command
	case gen.FullCommand():
		generate()
	}

}

func getConfig() (Config, error) {
	var configData Config

	if *configFile != nil {
		data, err := ioutil.ReadAll(*configFile)
		if err != nil {
			return configData, fmt.Errorf("Error reading configuration: %s", err)
		}

		err = json.Unmarshal(data, &configData)
		if err != nil {
			fmt.Println(err)
			return configData, err
		}
	} else {
		configData.Name = append(configData.Name, "User")
		configData.OrgName = "Chain33"
	}
	return configData, nil
}

func generateUsers(baseDir string, users []string, orgName string) {
	fmt.Printf("generateUsers\n")
	fmt.Println(baseDir)

	os.RemoveAll(baseDir)
	caDir := filepath.Join(baseDir, "cacerts")

	signCA, err := ca.NewCA(caDir, commonName)
	if err != nil {
		fmt.Printf("Error generating signCA", err.Error())
		os.Exit(1)
	}

	generateNodes(baseDir, users, signCA, orgName)
}

func generate() {
	conf, err := getConfig()
	if err != nil {
		fmt.Println("Configuation error")
		return
	}

	fmt.Println(conf.Name)
	fmt.Println(conf.OrgName)
	generateUsers(*outputDir, conf.Name, conf.OrgName)
}

func generateNodes(baseDir string, names []string, signCA *ca.CA, orgName string) {
	for _, name := range names {
		userDir := filepath.Join(baseDir, name)
		fileName := fmt.Sprintf("%s@%s", name, orgName)
		err := msp.GenerateLocalMSP(userDir, fileName, signCA)
		if err != nil {
			fmt.Printf("Error generating local MSP")
			os.Exit(1)
		}
	}
}
