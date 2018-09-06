package types

//game action ty
const (
	GameActionCreate = iota + 1
	GameActionMatch
	GameActionCancel
	GameActionClose
)

const (
	PackageName = "gitlab.33.cn.game"
	GameX       = "game"
)

var (
	ExecerGame = []byte(GameX)
)
