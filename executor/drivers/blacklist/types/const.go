package types

import (
	"errors"
)

const Fee = 1e6
const SecretLen = 32
const DefaultAmount = 1e11

var (
	ErrNotFound        = errors.New("ErrNotFound")
	CreditInsufficient = errors.New("Insufficient integration, please recharge.")
	ErrQueryNotSupport = errors.New("ErrQueryNotSupport")
)

const ConnIp = "localhost"
const AddCredit = 10

//httpListen 返回的错误提示
const (
	ErrorParm   = "The request parameter is wrong, please check again."
	ErrorMethod = "The request method is wrong, please check again."
)
const (
	SUCESS = "SUCESS"
	FAIL   = "FAIL"
)

//创世地址可以无限向其他orgAddr放送积分
const GenesisAddr = "0x0000000000000000000000000000"

//方法名定义
const (
	FuncName_SubmitRecord      = "submitRecord"
	FuncName_QueryRecordById   = "queryRecordById"
	FuncName_CreateOrg         = "createOrg"
	FuncName_QueryOrgById      = "queryOrgById"
	FuncName_QueryRecordByName = "queryRecordByName"
	FuncName_QueryTxById       = "queryTxByTxId"
	FuncName_QueryTxByFromAddr = "queryTxByFromAddr"
	FuncName_QueryTxByToAddr   = "queryTxByToAddr"
	FuncName_DeleteRecord      = "deleteRecord"
	FuncName_Transfer          = "transfer"
	FuncName_RegisterUser      = "registerUser"
	FuncName_LoginCheck        = "loginCheck"
	FuncName_ModifyUserPwd     = "modifyUserPwd"
	FuncName_ResetUserPwd      = "resetUserPwd"
	FuncName_CheckKeyIsExsit   = "resetUserPwd"
)
const (
	SubmitRecordDoc = "这是通过提交record来获取积分！"
)
