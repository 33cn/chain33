package types

import "gitlab.33.cn/chain33/chain33/types"

//retrieve
const (
	RetrievePreapre = iota + 1
	RetrievePerform
	RetrieveBackup
	RetrieveCancel
)

// retrieve op
const (
	RetrieveActionPrepare = 1
	RetrieveActionPerform = 2
	RetrieveActionBackup  = 3
	RetrieveActionCancel  = 4
)

var (
	JRPCName       = "Retrieve"
	RetrieveX      = "retrieve"
	ExecerRetrieve = []byte(RetrieveX)

	actionName = map[string]int32{
		"Prepare": RetrieveActionPrepare,
		"Perform": RetrieveActionPerform,
		"Backup":  RetrieveActionBackup,
		"Cancel":  RetrieveActionCancel,
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerRetrieve)
}
