package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type JsonClient struct {
	url string
}

func NewJsonClient(url string) (*JsonClient, error) {
	return &JsonClient{url}, nil
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

func (client *JsonClient) Call(method string, params, resp interface{}) error {
	req := &clientRequest{}
	req.Method = method
	req.Params[0] = params
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	log.Debug("request JsonStr", string(data), "")
	postresp, err := http.Post(client.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer postresp.Body.Close()
	b, err := ioutil.ReadAll(postresp.Body)
	if err != nil {
		return err
	}
	log.Debug("response", string(b), "")
	cresp := &clientResponse{}
	err = json.Unmarshal(b, &cresp)
	if err != nil {
		return err
	}
	if cresp.Error != nil || cresp.Result == nil {
		x, ok := cresp.Error.(string)
		if !ok {
			return fmt.Errorf("invalid error %v", cresp.Error)
		}
		if x == "" {
			x = "unspecified error"
		}
		return fmt.Errorf(x)
	}
	return json.Unmarshal(*cresp.Result, resp)
}
