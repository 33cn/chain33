package types

import (
	"errors"
)

var (
	ErrParaVoteBaseIndex = errors.New("ErrParaVoteBaseIndex")
)

const (
	//在平行链上保存节点参与共识的数据
	TyLogParacrossVote = 653
)
