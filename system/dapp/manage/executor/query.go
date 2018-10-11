package executor

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Manage) Query_GetConfigItem(in *types.ReqString) (types.Message, error) {
	// Load config from state db
	value, err := c.GetStateDB().Get([]byte(types.ManageKey(in.Data)))
	if err != nil {
		clog.Info("modifyConfig", "get db key", "not found")
		value = nil
	}
	if value == nil {
		value, err = c.GetStateDB().Get([]byte(types.ConfigKey(in.Data)))
		if err != nil {
			clog.Info("modifyConfig", "get db key", "not found")
			value = nil
		}
	}

	var reply types.ReplyConfig
	reply.Key = in.Data

	var item types.ConfigItem
	if value != nil {
		err = types.Decode(value, &item)
		if err != nil {
			clog.Error("modifyConfig", "get db key", in.Data)
			return nil, err // types.ErrBadConfigValue
		}
		reply.Value = fmt.Sprint(item.GetArr().Value)
	} else { // if config item not exist
		reply.Value = ""
	}
	clog.Info("manage  Query", "key ", in.Data)

	return &reply, nil
}
