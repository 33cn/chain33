package executor

import (
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (g *PokerBull) Query_QueryGameListByIds(in *pkt.QueryPBGameInfos) (types.Message, error) {
	return Infos(g.GetStateDB(), in)
}

func (g *PokerBull) Query_QueryGameById(in *pkt.QueryPBGameInfo) (types.Message, error) {
	game, err := readGame(g.GetStateDB(), in.GetGameId())
	if err != nil {
		return nil, err
	}
	return &pkt.ReplyPBGame{game}, nil
}

func (g *PokerBull) Query_QueryGameByAddr(in *pkt.QueryPBGameInfo) (types.Message, error) {
	gameIds, err := getGameListByAddr(g.GetLocalDB(), in.Addr, in.Index)
	if err != nil {
		return nil, err
	}

	return gameIds, nil
}

func (g *PokerBull) Query_QueryGameByStatus(in *pkt.QueryPBGameInfo) (types.Message, error) {
	gameIds, err := getGameListByStatus(g.GetLocalDB(), in.Status, in.Index)
	if err != nil {
		return nil, err
	}

	return gameIds, nil
}
