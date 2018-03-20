package types
import (
	"errors"
)

const Fee = 1e6
const SecretLen = 32
const DefaultAmount = 1e11
var (
	ErrNotFound                   = errors.New("ErrNotFound")
	CreditInsufficient            = errors.New("Insufficient integration, please recharge.")
	ErrQueryNotSupport           = errors.New("ErrQueryNotSupport")
)
const ConnIp="localhost"
const AddCredit =10
const  (
	TRUE="true"
	FALSE="false"
	//创世地址可以无限向其他orgAddr放送积分
	GenesisAddr="0x0000000000000000000000000000"
	SubmitRecord = "submitRecord"
	QueryRecordById = "queryRecordById"
	CreateOrg="createOrg"
	QueryOrgById="queryOrgById"
	QueryRecordByName="queryRecordByName"
	QueryTxById="queryTxByTxId"
	QueryTxByFromAddr="queryTxByFromAddr"
	QueryTxByToAddr="queryTxByToAddr"
	DeleteRecord="deleteRecord"
	Transfer="transfer"
	RegisterUser="registerUser"
    LoginCheck="loginCheck"
)
const(
	SubmitRecordDoc="这是通过提交record来获取积分！"
)