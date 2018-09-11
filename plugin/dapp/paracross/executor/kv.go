package executor

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/types"
)

var (
	title            string
	titleHeight      string
	configNodes      string
	localTx          string
	localTitle       string
	localTitleHeight string
)

func setPrefix() {
	title = "mavl-paracross-title-"
	titleHeight = "mavl-paracross-titleHeight-"
	configNodes = "paracross-nodes-"
	localTx = "paracross-titleHeightAddr-"
	localTitle = "paracross-title-"
	localTitleHeight = "paracross-titleHeight-"
}

func calcTitleKey(t string) []byte {
	return []byte(fmt.Sprintf(title+"%s", t))
}

func calcTitleHeightKey(title string, height int64) []byte {
	return []byte(fmt.Sprintf(titleHeight+"%s-%d", title, height))
}

func calcConfigNodesKey(title string) []byte {
	key := configNodes + title
	return []byte(types.ManageKey(key))
}

func calcLocalTxKey(title string, height int64, addr string) []byte {
	return []byte(fmt.Sprintf(localTx+"%s-%012-%s", title, height, addr))
}

func calcLocalTitleKey(title string) []byte {
	return []byte(fmt.Sprintf(localTitle+"%s", title))
}

func calcLocalTitleHeightKey(title string, height int64) []byte {
	return []byte(fmt.Sprintf(localTitle+"%s-%012d", title, height))
}

func calcLocalTitlePrefix() []byte {
	return []byte(localTitle)
}
