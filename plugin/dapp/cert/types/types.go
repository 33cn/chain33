package types

import "gitlab.33.cn/chain33/chain33/types"

//cert
const (
	CertActionNew    = 1
	CertActionUpdate = 2
	CertActionNormal = 3

	SignNameAuthECDSA = "auth_ecdsa"
	SignNameAuthSM2   = "auth_sm2"

	AUTH_ECDSA = 257
	AUTH_SM2   = 258
)

var mapSignType2name = map[int]string{
	AUTH_ECDSA: SignNameAuthECDSA,
	AUTH_SM2:   SignNameAuthSM2,
}

var mapSignName2Type = map[string]int{
	SignNameAuthECDSA: AUTH_ECDSA,
	SignNameAuthSM2:   AUTH_SM2,
}

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

func (base *CertType) GetCryptoDriver(ty int) (string, error) {
	if name, ok := mapSignType2name[ty]; ok {
		return name, nil
	}
	return "", types.ErrNotSupport
}

func (base *CertType) GetCryptoType(name string) (int, error) {
	if ty, ok := mapSignName2Type[name]; ok {
		return ty, nil
	}
	return 0, types.ErrNotSupport
}
