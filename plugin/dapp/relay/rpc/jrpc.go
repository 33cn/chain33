package rpc

import (
	"encoding/hex"
)

type Jrpc struct {
	cli channelClient
}

//type Grpc struct {
//	channelClient
//}

func (c *Jrpc) CreateRawRelayOrderTx(in *RelayOrderTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayOrderTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayAcceptTx(in *RelayAcceptTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayAcceptTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Jrpc) CreateRawRelayRevokeTx(in *RelayRevokeTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Jrpc) CreateRawRelayConfirmTx(in *RelayConfirmTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayConfirmTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Jrpc) CreateRawRelayVerifyBTCTx(in *RelayVerifyBTCTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayVerifyBTCTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelaySaveBTCHeadTx(in *RelaySaveBTCHeadTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelaySaveBTCHeadTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)

	return nil
}
