package executor

import (
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (g *Game) Query_QueryGameListByIds(in *gt.QueryGameInfos) (types.Message, error) {
	return Infos(g.GetStateDB(), in)
}

func (g *Game) Query_QueryGameListCount(in *gt.QueryGameListCount) (types.Message, error) {
	return QueryGameListCount(g.GetStateDB(), in)
}

func (g *Game) Query_QueryGameListByStatusAndAddr(in *gt.QueryGameListByStatusAndAddr) (types.Message, error) {
	return List(g.GetLocalDB(), g.GetStateDB(), in)
}

func (g *Game) Query_QueryGameById(in *gt.QueryGameInfo) (types.Message, error) {
	game, err := readGame(g.GetStateDB(), in.GetGameId())
	if err != nil {
		return nil, err
	}
	return &gt.ReplyGame{game}, nil
}
