package eth_rpc

import "errors"

var (
	Err_AddressFormat =errors.New("invalid argument 0: json: cannot unmarshal hex string without 0x prefix into Go value of type common.Address")
)
