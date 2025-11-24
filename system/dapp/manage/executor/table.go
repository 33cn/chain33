package executor

import (
	"fmt"

	"github.com/33cn/chain33/common/db"

	"github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/system/dapp"
	mty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
)

/*
table  struct
data:  manage config
index: status, addr
*/

var configOpt = &table.Option{
	Prefix:  "LODB-manage",
	Name:    "config",
	Primary: "heightindex",
	Index:   []string{"addr", "status", "addr_status"},
}

// NewConfigTable 新建表
func NewConfigTable(kvdb db.KV) *table.Table {
	rowmeta := NewConfigRow()
	newTable, err := table.NewTable(rowmeta, kvdb, configOpt)
	if err != nil {
		panic(err)
	}
	return newTable
}

// ConfigRow table meta 结构
type ConfigRow struct {
	*mty.ConfigStatus
}

// NewConfigRow 新建一个meta 结构
func NewConfigRow() *ConfigRow {
	return &ConfigRow{ConfigStatus: &mty.ConfigStatus{}}
}

// CreateRow 新建数据行(注意index 数据一定也要保存到数据中,不能就保存heightindex)
func (r *ConfigRow) CreateRow() *table.Row {
	return &table.Row{Data: &mty.ConfigStatus{}}
}

// SetPayload 设置数据
func (r *ConfigRow) SetPayload(data types.Message) error {
	if d, ok := data.(*mty.ConfigStatus); ok {
		r.ConfigStatus = d
		return nil
	}
	return types.ErrTypeAsset
}

// Get 按照indexName 查询 indexValue
func (r *ConfigRow) Get(key string) ([]byte, error) {
	if key == "heightindex" {
		return []byte(dapp.HeightIndexStr(r.Height, int64(r.Index))), nil
	} else if key == "status" {
		return []byte(fmt.Sprintf("%2d", r.Status)), nil
	} else if key == "addr" {
		return []byte(r.Proposer), nil
	} else if key == "addr_status" {
		return []byte(fmt.Sprintf("%s:%2d", r.Proposer, r.Status)), nil
	}
	return nil, types.ErrNotFound
}
