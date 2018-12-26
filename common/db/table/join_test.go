package table

import (
	"testing"

	"github.com/33cn/chain33/common/db/table/proto"

	"github.com/33cn/chain33/types"
)

func TestJoin(t *testing.T) {

}

/*
table  game
data:  Game
index: addr,status,index,type
*/

var optgame = &Option{
	Prefix:  "LODB",
	Name:    "game",
	Primary: "gameID",
	Index:   []string{"status"},
}

//GameRow table meta 结构
type GameRow struct {
	*proto.Game
}

//NewGameRow 新建一个meta 结构
func NewGameRow() *GameRow {
	return &GameRow{Game: &proto.Game{}}
}

//CreateRow 新建数据行(注意index 数据一定也要保存到数据中,不能就保存eventid)
func (tx *GameRow) CreateRow() *Row {
	return &Row{Data: &proto.Game{}}
}

//SetPayload 设置数据
func (tx *GameRow) SetPayload(data types.Message) error {
	if txdata, ok := data.(*proto.Game); ok {
		tx.Game = txdata
		return nil
	}
	return types.ErrTypeAsset
}

//Get 按照indexName 查询 indexValue
func (tx *GameRow) Get(key string) ([]byte, error) {
	return nil, types.ErrNotFound
}

/*
table  struct
data:  GameAddr
index: addr,status,index,type
*/

var optgameaddr = &Option{
	Prefix:  "LODB",
	Name:    "gameaddr",
	Primary: "txhash",
	Index:   []string{"gameID", "addr"},
}

//GameAddrRow table meta 结构
type GameAddrRow struct {
	*proto.GameAddr
}

//NewGameAddrRow 新建一个meta 结构
func NewGameAddrRow() *GameAddrRow {
	return &GameAddrRow{GameAddr: &proto.GameAddr{}}
}

//CreateRow 新建数据行(注意index 数据一定也要保存到数据中,不能就保存eventid)
func (tx *GameAddrRow) CreateRow() *Row {
	return &Row{Data: &proto.Game{}}
}

//SetPayload 设置数据
func (tx *GameAddrRow) SetPayload(data types.Message) error {
	if txdata, ok := data.(*proto.GameAddr); ok {
		tx.GameAddr = txdata
		return nil
	}
	return types.ErrTypeAsset
}

//Get 按照indexName 查询 indexValue
func (tx *GameAddrRow) Get(key string) ([]byte, error) {
	return nil, types.ErrNotFound
}

/*
joinopt = &OptionJoin{
	Prefix:  "LODB",
	Index: []string{"addr-status"} //两个索引的联合（addr 必须在 table2 中， status 必须在 table1 中）
}
tablejoin := NewJoinTable(table2, table1, joinopt)
tablejoin.Left() //table2
tablejoin.Right() //table1
//查询结构
tablejoin.Save() //save table2, table1, tablejoin
//tablejoin 不能更新
tablejoin.Query()
*/
