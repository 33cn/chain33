// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

type JSONClient struct {
	url    string
	prefix string
}

func addPrefix(prefix, name string) string {
	if strings.Contains(name, ".") {
		return name
	}
	return prefix + "." + name
}

func NewJSONClient(url string) (*JSONClient, error) {
	return &JSONClient{url: url, prefix: "Chain33"}, nil
}

func New(prefix, url string) (*JSONClient, error) {
	return &JSONClient{url: url, prefix: prefix}, nil
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	Id     uint64         `json:"id"`
}

type clientResponse struct {
	Id     uint64           `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func (client *JSONClient) Call(method string, params, resp interface{}) error {
	method = addPrefix(client.prefix, method)
	req := &clientRequest{}
	req.Method = method
	req.Params[0] = params
	data, err := json.MarshalIndent(req, "", "\t")
	if err != nil {
		return err
	}
	//println("request JsonStr", string(data), "")
	postresp, err := http.Post(client.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer postresp.Body.Close()
	b, err := ioutil.ReadAll(postresp.Body)
	if err != nil {
		return err
	}
	//println("response", string(b), "")
	cresp := &clientResponse{}
	err = json.Unmarshal(b, &cresp)
	if err != nil {
		return err
	}
	if cresp.Error != nil /*|| cresp.Result == nil*/ {
		x, ok := cresp.Error.(string)
		if !ok {
			return fmt.Errorf("invalid error %v", cresp.Error)
		}
		if x == "" {
			x = "unspecified error"
		}
		return fmt.Errorf(x)
	}
	if cresp.Result == nil {
		return types.ErrEmpty
	} else {
		if msg, ok := resp.(proto.Message); ok {
			var str json.RawMessage
			err = json.Unmarshal(*cresp.Result, &str)
			if err != nil {
				return err
			}
			b, err := str.MarshalJSON()
			if err != nil {
				return err
			}
			return types.JsonToPB(b, msg)
		}
		return json.Unmarshal(*cresp.Result, resp)
	}
}
