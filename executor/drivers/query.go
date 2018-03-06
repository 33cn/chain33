package drivers

import (
	"errors"
	"fmt"

	"code.aliyun.com/chain33/chain33/types"
)

//用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
//地址下面所有的交易
func calcTxAddrHashKey(addr string, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrHash:%s:%s", addr, heightindex))
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
//地址下面某个分类的交易
func calcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrDirHash:%s:%d:%s", addr, flag, heightindex))
}

// 通过addr前缀查找本地址参与的所有交易
//查询交易默认放到：coins 中查询
func (n *ExecBase) GetTxsByAddr(addr *types.ReqAddr) (types.Message, error) {
	db := n.GetQueryDB()
	var prefix []byte
	var key []byte
	var txinfos [][]byte
	//取最新的交易hash列表
	if addr.Flag == 0 { //所有的交易hash列表
		prefix = calcTxAddrHashKey(addr.GetAddr(), "")
	} else if addr.Flag > 0 { //from的交易hash列表
		prefix = calcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, "")
	} else {
		return nil, errors.New("Flag unknow!")
	}
	blog.Error("GetTxsByAddr", "height", addr.GetHeight())
	if addr.GetHeight() == -1 {
		txinfos = db.IteratorScanFromLast(prefix, addr.Count, addr.Direction)
		if len(txinfos) == 0 {
			return nil, errors.New("does not exist tx!")
		}
	} else { //翻页查找指定的txhash列表
		blockheight := addr.GetHeight()*types.MaxTxsPerBlock + int64(addr.GetIndex())
		heightstr := fmt.Sprintf("%018d", blockheight)
		if addr.Flag == 0 {
			key = calcTxAddrHashKey(addr.GetAddr(), heightstr)
		} else if addr.Flag > 0 { //from的交易hash列表
			key = calcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, heightstr)
		} else {
			return nil, errors.New("Flag unknow!")
		}
		txinfos = db.IteratorScan(prefix, key, addr.Count, addr.Direction)
		if len(txinfos) == 0 {
			return nil, errors.New("does not exist tx!")
		}
	}
	var replyTxInfos types.ReplyTxInfos
	replyTxInfos.TxInfos = make([]*types.ReplyTxInfo, len(txinfos))
	for index, txinfobyte := range txinfos {
		var replyTxInfo types.ReplyTxInfo
		err := types.Decode(txinfobyte, &replyTxInfo)
		if err != nil {
			blog.Error("GetTxsByAddr proto.Unmarshal!", "err:", err)
			return nil, err
		}
		replyTxInfos.TxInfos[index] = &replyTxInfo
	}
	return &replyTxInfos, nil
}
