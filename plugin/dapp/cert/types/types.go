package types

import "gitlab.33.cn/chain33/chain33/types"

//cert
const (
	CertActionNew    = 1
	CertActionUpdate = 2
	CertActionNormal = 3
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerCert)

	// init executor type
	types.RegistorExecutor(CertX, NewType())
}

type CertType struct {
	types.ExecTypeBase
}

func NewType() *CertType {
	c := &CertType{}
	c.SetChild(c)
	return c
}

func (b *CertType) GetPayload() types.Message {
	return &CertAction{}
}

func (b *CertType) GetName() string {
	return CertX
}

func (b *CertType) GetLogMap() map[int64]*types.LogInfo {
	return nil
}

func (b *CertType) GetTypeMap() map[string]int32 {
	return actionName
}
