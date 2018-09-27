package executor

import (
	"github.com/pkg/errors"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Paracross) ParacrossGetHeight(title string) (types.Message, error) {
	ret, err := getTitle(c.GetStateDB(), calcTitleKey(title))
	if err != nil {
		return nil, errors.Cause(err)
	}
	return ret, nil
}

func (c *Paracross) ParacrossListTitles() (types.Message, error) {
	return listLocalTitles(c.GetLocalDB())
}

func listLocalTitles(db dbm.KVDB) (types.Message, error) {
	prefix := calcLocalTitlePrefix()
	res, err := db.List(prefix, []byte(""), 0, 1)
	if err != nil {
		return nil, err
	}
	var resp pt.RespParacrossTitles
	for _, r := range res {
		var st pt.ReceiptParacrossDone
		err = types.Decode(r, &st)
		if err != nil {
			panic(err)
		}
		resp.Titles = append(resp.Titles, &st)
	}
	return &resp, nil
}

func loadLocalTitle(db dbm.KV, title string, height int64) (types.Message, error) {
	key := calcLocalTitleHeightKey(title, height)
	res, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	var resp pt.ReceiptParacrossDone
	err = types.Decode(res, &resp)
	if err != nil {
		panic(err)
	}
	return &resp, nil
}

func (c *Paracross) ParacrossGetTitleHeight(title string, height int64) (types.Message, error) {
	return loadLocalTitle(c.GetLocalDB(), title, height)
}

func (c *Paracross) ParacrossGetAssetTxResult(hash []byte) (types.Message, error) {
	if len(hash) == 0 {
		return nil, types.ErrInputPara
	}

	key := calcLocalAssetKey(hash)
	value, err := c.GetLocalDB().Get(key)
	if err != nil {
		return nil, err
	}

	var result pt.ParacrossAsset
	err = types.Decode(value, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
