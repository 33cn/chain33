package blacklist
import (
	"errors"
)

const fee = 1e6
const secretLen = 32
const defaultAmount = 1e11
var (
	ErrNotFound                   = errors.New("ErrNotFound")
	CreditInsufficient            = errors.New("Insufficient integration, please recharge.")
	ErrQueryNotSupport           = errors.New("ErrQueryNotSupport")
)
const AddCredit =10
const  (
	//创世地址可以无限向其他orgAddr放送积分
	GenesisAddr="0x0000000000000000000000000000"
	SubmitRecord = "submitRecord"
	QueryRecord = "queryRecord"
	CreateOrg="createOrg"
	QueryOrg="queryOrg"
	QueryRecordByName="queryRecordByName"
	DeleteRecord="deleteRecord"
)
const(
	SubmitRecordDoc="这是通过提交record来获取积分！"
)