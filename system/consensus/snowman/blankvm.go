package snowman

import (
	"context"
	"time"

	"github.com/ava-labs/avalanchego/api/health"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/version"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
)

var (
	_ common.AppHandler    = (*blankVM)(nil)
	_ health.Checker       = (*blankVM)(nil)
	_ validators.Connector = (*blankVM)(nil)
)

type blankVM struct {
	//ctx        *snow.Context
}

// CrossChainAppRequestFailed blank
func (*blankVM) CrossChainAppRequestFailed(context.Context, ids.ID, uint32) error {
	recordUnimplementedError("CrossChainAppRequestFailed")
	return nil
}

// CrossChainAppRequest blank
func (*blankVM) CrossChainAppRequest(context.Context, ids.ID, uint32, time.Time, []byte) error {
	recordUnimplementedError("CrossChainAppRequest")
	return nil
}

// CrossChainAppResponse blank
func (*blankVM) CrossChainAppResponse(context.Context, ids.ID, uint32, []byte) error {
	recordUnimplementedError("CrossChainAppResponse")
	return nil
}

// AppRequestFailed blank
func (*blankVM) AppRequestFailed(context.Context, ids.NodeID, uint32) error {
	// This VM currently only supports gossiping of txs, so there are no
	// requests.
	recordUnimplementedError("AppRequestFailed")
	return nil
}

// AppRequest blank
func (*blankVM) AppRequest(context.Context, ids.NodeID, uint32, time.Time, []byte) error {
	// This VM currently only supports gossiping of txs, so there are no
	// requests.
	recordUnimplementedError("AppRequest")
	return nil
}

// AppResponse blank
func (*blankVM) AppResponse(context.Context, ids.NodeID, uint32, []byte) error {
	// This VM currently only supports gossiping of txs, so there are no
	// requests.
	recordUnimplementedError("AppResponse")
	return nil
}

// AppGossip blank
func (*blankVM) AppGossip(_ context.Context, nodeID ids.NodeID, msgBytes []byte) error {
	recordUnimplementedError("AppGossip")
	return nil
}

// GossipTx blank
func (*blankVM) GossipTx(tx *txs.Tx) error {

	recordUnimplementedError("GossipTx")
	return nil
}

// HealthCheck health check
func (*blankVM) HealthCheck(ctx context.Context) (interface{}, error) {
	recordUnimplementedError("HealthCheck")
	return nil, nil
}

// Connected notify connected
func (*blankVM) Connected(_ context.Context, nodeID ids.NodeID, _ *version.Application) error {
	recordUnimplementedError("Connected")
	return nil
}

// Disconnected notify disconnected
func (*blankVM) Disconnected(_ context.Context, nodeID ids.NodeID) error {
	recordUnimplementedError("Disconnected")
	return nil
}

// CreateHandlers makes new http handlers that can handle API calls
func (*blankVM) CreateHandlers(context.Context) (map[string]*common.HTTPHandler, error) {
	recordUnimplementedError("CreateHandlers")
	return nil, nil
}

// CreateStaticHandlers makes new http handlers that can handle API calls
func (*blankVM) CreateStaticHandlers(context.Context) (map[string]*common.HTTPHandler, error) {
	recordUnimplementedError("CreateStaticHandlers")
	return nil, nil
}

// Version returns the version of the VM.
func (*blankVM) Version(context.Context) (string, error) {
	return "blank-version", nil
}

func recordUnimplementedError(funcName string) {

	snowLog.Error("Call unimplemented function", "func", funcName, "stack", getStack())
}
