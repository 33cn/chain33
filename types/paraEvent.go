package types

import (
	"errors"
)

const (
	ParaX = "paracross"
)

var (
	ExecerPara      = []byte(ParaX)
	TxGroupMaxCount = 20

	ErrTicketLocked = errors.New("ErrTicketLocked")
)

func GetTitle() string {
	return title
}
