package table

import (
	"fmt"
	"testing"

	protodata "github.com/33cn/chain33/common/db/table/proto"
	"github.com/33cn/chain33/util"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/33cn/chain33/types"
)

func TestJoin(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	table1, err := NewTable(NewGameRow(), kvdb, optgame)
	assert.Nil(t, err)
	table2, err := NewTable(NewGameAddrRow(), kvdb, optgameaddr)
	assert.Nil(t, err)
	tablejoin, err := NewJoinTable(table2, table1, []string{"addr#status", "#status"})
	assert.Nil(t, err)
	assert.Equal(t, tablejoin.GetLeft(), table2)                //table2
	assert.Equal(t, tablejoin.GetRight(), table1)               //table1
	assert.Equal(t, tablejoin.MustGetTable("gameaddr"), table2) //table2
	assert.Equal(t, tablejoin.MustGetTable("game"), table1)     //table1

	rightdata := &protodata.Game{GameID: "gameid1", Status: 1}
	tablejoin.MustGetTable("game").Replace(rightdata)
	leftdata := &protodata.GameAddr{GameID: "gameid1", Addr: "addr1", Txhash: "hash1"}
	tablejoin.MustGetTable("gameaddr").Replace(leftdata)
	kvs, err := tablejoin.Save()
	assert.Nil(t, err)
	assert.Equal(t, 7, len(kvs))
	util.SaveKVList(ldb, kvs)
	//query table
	//每个表的查询，用 tablejoin.MustGetTable("gameaddr")
	//join query 用 tablejoin.Query
	rows, err := tablejoin.ListIndex("addr#status", JoinKey([]byte("addr1"), []byte("1")), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Left, leftdata))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Right, rightdata))

	rows, err = tablejoin.ListIndex("#status", JoinKey(nil, []byte("1")), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Left, leftdata))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Right, rightdata))

	rightdata = &protodata.Game{GameID: "gameid1", Status: 2}
	tablejoin.MustGetTable("game").Replace(rightdata)
	kvs, err = tablejoin.Save()
	assert.Nil(t, err)
	assert.Equal(t, 7, len(kvs))
	util.SaveKVList(ldb, kvs)
	rows, err = tablejoin.ListIndex("addr#status", JoinKey([]byte("addr1"), []byte("2")), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Left, leftdata))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Right, rightdata))

	rows, err = tablejoin.ListIndex("#status", JoinKey(nil, []byte("2")), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Left, leftdata))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Right, rightdata))

	rightdata = &protodata.Game{GameID: "gameid1", Status: 2}
	tablejoin.MustGetTable("game").Replace(rightdata)
	kvs, err = tablejoin.Save()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(kvs))

	leftdata = &protodata.GameAddr{GameID: "gameid1", Addr: "addr2", Txhash: "hash1"}
	tablejoin.MustGetTable("gameaddr").Replace(leftdata)
	kvs, err = tablejoin.Save()
	assert.Nil(t, err)
	assert.Equal(t, 5, len(kvs))
	util.SaveKVList(ldb, kvs)

	//改回到全部是1的情况
	rightdata = &protodata.Game{GameID: "gameid1", Status: 1}
	tablejoin.MustGetTable("game").Replace(rightdata)
	leftdata = &protodata.GameAddr{GameID: "gameid1", Addr: "addr1", Txhash: "hash1"}
	tablejoin.MustGetTable("gameaddr").Replace(leftdata)
	kvs, err = tablejoin.Save()
	assert.Nil(t, err)
	assert.Equal(t, 10, len(kvs))
	util.SaveKVList(ldb, kvs)
	rows, err = tablejoin.ListIndex("addr#status", JoinKey([]byte("addr1"), []byte("1")), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Left, leftdata))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Right, rightdata))

	rows, err = tablejoin.ListIndex("#status", JoinKey(nil, []byte("1")), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Left, leftdata))
	assert.Equal(t, true, proto.Equal(rows[0].Data.(*JoinData).Right, rightdata))
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
	*protodata.Game
}

//NewGameRow 新建一个meta 结构
func NewGameRow() *GameRow {
	return &GameRow{Game: &protodata.Game{}}
}

//CreateRow 新建数据行(注意index 数据一定也要保存到数据中,不能就保存eventid)
func (tx *GameRow) CreateRow() *Row {
	return &Row{Data: &protodata.Game{}}
}

//SetPayload 设置数据
func (tx *GameRow) SetPayload(data types.Message) error {
	if txdata, ok := data.(*protodata.Game); ok {
		tx.Game = txdata
		return nil
	}
	return types.ErrTypeAsset
}

//Get 按照indexName 查询 indexValue
func (tx *GameRow) Get(key string) ([]byte, error) {
	if key == "gameID" {
		return []byte(tx.GameID), nil
	} else if key == "status" {
		return []byte(fmt.Sprint(tx.Status)), nil
	}
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
	*protodata.GameAddr
}

//NewGameAddrRow 新建一个meta 结构
func NewGameAddrRow() *GameAddrRow {
	return &GameAddrRow{GameAddr: &protodata.GameAddr{}}
}

//CreateRow 新建数据行(注意index 数据一定也要保存到数据中,不能就保存eventid)
func (tx *GameAddrRow) CreateRow() *Row {
	return &Row{Data: &protodata.GameAddr{}}
}

//SetPayload 设置数据
func (tx *GameAddrRow) SetPayload(data types.Message) error {
	if txdata, ok := data.(*protodata.GameAddr); ok {
		tx.GameAddr = txdata
		return nil
	}
	return types.ErrTypeAsset
}

//Get 按照indexName 查询 indexValue
func (tx *GameAddrRow) Get(key string) ([]byte, error) {
	if key == "gameID" {
		return []byte(tx.GameID), nil
	} else if key == "addr" {
		return []byte(tx.Addr), nil
	} else if key == "txhash" {
		return []byte(tx.Txhash), nil
	}
	return nil, types.ErrNotFound
}
