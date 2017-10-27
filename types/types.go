package types

import proto "github.com/golang/protobuf/proto"
import "code.aliyun.com/chain33/chain33/common"

func (tx *Transaction) Hash() []byte {
	copytx := *tx
	copytx.Signature = nil
	data, err := proto.Marshal(copytx)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)[:]
}
