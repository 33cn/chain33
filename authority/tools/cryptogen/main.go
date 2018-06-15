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
	"text/template"

	"gopkg.in/alecthomas/kingpin.v2"

	"bytes"
	"io/ioutil"

	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/ca"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/msp"
)

const (
	commonName              = "ca"
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
		//str := []string{"User"}
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
	//userDir := filepath.Join(baseDir, "users")

	signCA, err := ca.NewCA(caDir, commonName)
	if err != nil {
		fmt.Printf("Error generating signCA")
		os.Exit(1)
	}

	generateNodes(baseDir, users, signCA, orgName)
	//err = msp.GenerateVerifyingMSP(mspDir, signCA, tlsCA)
	//if err != nil {
	//	fmt.Printf("Error generating MSP for org %s:\n%v\n", orgName, err)
	//	os.Exit(1)
	//}

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

func parseTemplate(input string, data interface{}) (string, error) {

	t, err := template.New("parse").Parse(input)
	if err != nil {
		return "", fmt.Errorf("Error parsing template: %s", err)
	}

	output := new(bytes.Buffer)
	err = t.Execute(output, data)
	if err != nil {
		return "", fmt.Errorf("Error executing template: %s", err)
	}

	return output.String(), nil
}

func generateNodes(baseDir string, names []string, signCA *ca.CA, orgName string) {

	for _, name := range names {
		userDir := filepath.Join(baseDir, name)
		//add org here, reference:NewFileCertStore
		fileName := fmt.Sprintf("%s@%s", name, orgName)
		err := msp.GenerateLocalMSP(userDir, fileName, signCA)
		if err != nil {
			fmt.Printf("Error generating local MSP")
			os.Exit(1)
		}
	}
}
