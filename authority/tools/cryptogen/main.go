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
	"io"
	"os"
	"path/filepath"
	"text/template"
	//"gopkg.in/yaml.v2"

	"gopkg.in/alecthomas/kingpin.v2"

	"bytes"
	"io/ioutil"

	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/ca"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/metadata"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/msp"
)

const (
	userBaseName            = "User"
	adminBaseName           = "Admin"
	defaultHostnameTemplate = "{{.Prefix}}{{.Index}}"
	defaultCNTemplate       = "{{.Hostname}}.{{.Domain}}"
	commonName              = "ca"
)

type Config struct {
	Name    []string
	OrgName string
}

//command line flags
var (
	app = kingpin.New("cryptogen", "Utility for generating Hyperledger Fabric key material")

	gen        = app.Command("generate", "Generate key material")
	outputDir  = gen.Flag("output", "The output directory in which to place artifacts").Default("../../../cmd/chain33/authdir/crypto").String()
	configFile = gen.Flag("config", "The configuration template to use").File()
	userName   = gen.Flag("user", "The username for crypto").Default("user").String()

	showtemplate = app.Command("showtemplate", "Show the default configuration template")

	version = app.Command("version", "Show version information")
)

func main() {
	kingpin.Version("0.0.1")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	// "generate" command
	case gen.FullCommand():
		generate()

	// "version" command
	case version.FullCommand():
		printVersion()
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

		return configData, nil
	} else {
		panic("")
	}
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
	fmt.Println(conf.Name)
	fmt.Println(conf.OrgName)
	fmt.Println(err)
	if err == nil {
		generateUsers(*outputDir, conf.Name, conf.OrgName)
	}
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

func parseTemplateWithDefault(input, defaultInput string, data interface{}) (string, error) {

	// Use the default if the input is an empty string
	if len(input) == 0 {
		input = defaultInput
	}

	return parseTemplate(input, data)
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func printVersion() {
	fmt.Println(metadata.GetVersionInfo())
}
