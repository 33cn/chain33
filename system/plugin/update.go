package plugin

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

var (
	versionPrefix = "version"
)

// GetVersion plugin localdb Version
func GetVersion(kvdb dbm.KV, name string) (int, error) {
	value, err := kvdb.Get(versionKey(name))
	if err != nil && err != types.ErrNotFound {
		return 1, err
	}
	if err == types.ErrNotFound {
		return 1, nil
	}
	var v types.Int32
	err = types.Decode(value, &v)
	if err != nil {
		return 1, err
	}
	return int(v.Data), nil
}

// SetVersion set plugin db version
func SetVersion(kvdb dbm.KV, name string, version int) error {
	v := types.Int32{Data: int32(version)}
	x := types.Encode(&v)
	return kvdb.Set(versionKey(name), x)
}

func versionKey(name string) []byte {
	return append(types.LocalPluginPrefix, []byte(fmt.Sprintf("%s-%s", name, versionPrefix))...)
}
