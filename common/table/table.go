package table

import "github.com/33cn/chain33/types"

//RowMeta 定义行的操作
type RowMeta interface {
	CreateRow() types.Message
	SetPayload(types.Message)
	Get(key string) ([]byte, error)
}

type Row struct {
	Ty   int
	Data types.Message
}

//Table 定一个表格, 并且添加 primary key, index key
type Table struct {
	meta    RowMeta
	rows    []Row
	primary string
	index   []string
}

//NewTable  新建一个表格
//primary 可以为: auto, 由系统自动创建
//index 可以为nil
func NewTable(meta RowMeta, primary string, index []string) (*Table, error) {
	if len(index) > 16 {
		return nil, ErrTooManyIndex
	}
	if err := checkPrimaryKey(meta, primary); err != nil {
		return nil, err
	}
	return &Table{meta: meta, primary: primary, index: index}, nil
}

func checkPrimaryKey(meta RowMeta, primary string) error {
	if primary == "" {
		return ErrEmptyPrimaryKey
	}
	if primary != "auto" {
		_, err := meta.Get(primary)
		if err != nil {
			return err
		}
	}
	return nil
}

func (table *Table) checkIndex(data types.Message) error {
	table.meta.SetPayload(data)
	if err := checkPrimaryKey(table.meta, table.primary); err != nil {
		return err
	}
	for i := 0; i < len(table.index); i++ {
		_, err := table.meta.Get(table.index[i])
		if err != nil {
			return err
		}
	}
	return nil
}

//Add 在表格中添加一行
func (table *Table) Add(data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	table.rows = append(table.rows, data)
	return nil
}

//del 在表格中删除一行
