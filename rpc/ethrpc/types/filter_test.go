package types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
)

func Test_Filter(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	var blockHash = "0xc48e9635c678785927b05eb8ff5c8a8df898dd76c5b91329117ce8cbf429605a"
	var detail ctypes.TransactionDetail
	detail.Height = 85
	detail.Fromaddr = "0x495953a743ef169ec5d4ac7b5f786bf2bd56afd5"
	detail.Index = 0
	detail.Receipt = &ctypes.ReceiptData{Ty: 2,
		Logs: []*ctypes.ReceiptLog{
			{
				Ty:  605,
				Log: common.FromHex("0x0a20ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef0a20000000000000000000000000495953a743ef169ec5d4ac7b5f786bf2bd56afd50a200000000000000000000000006eb76c83ad12b865a43cfab57d5f2af1288dd51d12200000000000000000000000000000000000000000000000001bc16d674ec80000"),
			},
			{
				Ty:  605,
				Log: common.FromHex("0x0a208c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9250a20000000000000000000000000495953a743ef169ec5d4ac7b5f786bf2bd56afd50a200000000000000000000000006eb76c83ad12b865a43cfab57d5f2af1288dd51d12200000000000000000000000000000000000000000000000000000000000000000"),
			},
			{
				Ty:  605,
				Log: common.FromHex("0x0a20d0943372c08b438a88d4b39d77216901079eda9ca59d45349841c099083b68300a20000000000000000000000000495953a743ef169ec5d4ac7b5f786bf2bd56afd5128002000000000000000000000000000000000000000000000000000000000000001500000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000002a355000000000000000000000000000000000000000000000000000000000000004d01000000000000000200000000000000001bc16d674ec80000495953a743ef169ec5d4ac7b5f786bf2bd56afd52c22d3715e8e60db6fd784f208ceebedfeac515dcff6ef51b78c80ded64f0e4200000000000000000000000000000000000000"),
			},
			{Ty: 605,
				Log: common.FromHex("0x0a203df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd0a2000000000000000000000000000000000000000000000000000000000000000020a20000000000000000000000000495953a743ef169ec5d4ac7b5f786bf2bd56afd512200000000000000000000000000000000000000000000000001bc16d674ec80000"),
			},
			{
				Ty:  603,
				Log: common.FromHex("0x0a2a3078343935393533613734336566313639656335643461633762356637383662663262643536616664351a2a30783665623736633833616431326238363561343363666162353764356632616631323838646435316420bbc10a"),
			},
		},
	}
	detail.Blocktime = 1669357616
	detail.Tx = &ctypes.Transaction{
		Execer:  []byte("evm"),
		Payload: common.FromHex("0x10f0930918012a44a9059cbb00000000000000000000000070f657164e5b75689b64b7fd1fa275f334f28e180000000000000000000000000000000000000000000000000000000005f5e1003ae00266386165383230313461383530323534306265343030383330323439663039346561313263626533633430616438343762346636393465313939626339303730623562616335653538306238343461393035396362623030303030303030303030303030303030303030303030303730663635373136346535623735363839623634623766643166613237356633333466323865313830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303035663565313030383231373932613064343537376538346166643864353134633435326333373666653062353863643231626264623432303730663735356439363930356430313064323666323061613032343232313937616333663765343630386131306637316436633264616166623230313164323861343361613163623962626432323138353337366430626131422a307865613132636265336334306164383437623466363934653139396263393037306235626163356535"),
		Signature: &ctypes.Signature{
			Ty:        8452,
			Pubkey:    common.FromHex("0x04ba61adbb0d24d3bd2bf5be57cb2d4c506ea3fc077f18703b484edd23b629f285d0751aa90603c22c72816eccbd0667624557f1ac551592d70eabb2038234742a"),
			Signature: common.FromHex("0xfbf8d4a3728b44b677f6f7e7f59673faaa4141e01072f43b8bc5faa6b92ffa195a2b3c47110f15ad7bebfa25d60a31de190ff49276cce148f70c1bcacec45c6d01"),
		},
		Fee:     150000,
		Nonce:   330,
		Expire:  1669368416,
		To:      "0x6eb76c83ad12b865a43cfab57d5f2af1288dd51d",
		ChainID: 3999,
	}
	var details ctypes.TransactionDetails
	details.Txs = append(details.Txs, &detail)
	_, receipt, err := TxDetailsToEthReceipts(&details, common.BytesToHash(common.FromHex(blockHash)), cfg)
	if err != nil {
		t.Log(err)
		return
	}

	filterReceipt(t, receipt)

}
func filterReceipt(t *testing.T, receipt []*Receipt) {
	var blockHash = "0xc48e9635c678785927b05eb8ff5c8a8df898dd76c5b91329117ce8cbf429605a"
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	filter, err := NewFilter(nil, cfg, nil)
	assert.Nil(t, err)
	logs := filter.filterReceipt(receipt)
	assert.Equal(t, len(logs), 4)
	assert.Equal(t, logs[0].BlockHash.String(), blockHash)
	assert.Equal(t, logs[0].Data.String(), "0x0000000000000000000000000000000000000000000000001bc16d674ec80000")
	var option FilterQuery
	option.Address = "0xea12cbe3c40ad847b4f694e199bc9070b5bac5e5"
	option.Topics = append(option.Topics, "0x3df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd")
	//option.Topics = []common.Hash{{common.HexToHash("0x3df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd")}}
	filter, err = NewFilter(nil, cfg, &option)
	assert.Nil(t, err)
	logs = filter.filterReceipt(receipt)
	assert.Equal(t, logs[0].BlockHash.String(), blockHash)
	assert.Equal(t, logs[0].Data.String(), "0x0000000000000000000000000000000000000000000000001bc16d674ec80000")
	option.Topics = nil
	option.Topics = append(option.Topics, "0x3df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd")
	filter, err = NewFilter(nil, cfg, &option)
	logs = filter.filterReceipt(receipt)
	assert.Equal(t, logs[0].BlockHash.String(), blockHash)
	assert.Equal(t, logs[0].Data.String(), "0x0000000000000000000000000000000000000000000000001bc16d674ec80000")

	option.Address = "0xea12cbe3c40ad847b4f694e199bc9070b5bac5e5"
	topic := "0x3df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd"
	option.Topics = append(option.Topics, topic)
	filter, err = NewFilter(nil, cfg, &option)
	assert.Nil(t, err)
	logs = filter.filterReceipt(receipt)
	assert.Equal(t, 1, len(logs))
	var hashes []interface{}
	hashes = append(hashes, "0x3df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd", "0xd0943372c08b438a88d4b39d77216901079eda9ca59d45349841c099083b6830")
	//option.Topics=nil
	option.Topics = hashes
	t.Log("topics", option.Topics)
	filter, err = NewFilter(nil, cfg, &option)
	assert.Nil(t, err)

	logs = filter.filterReceipt(receipt)
	assert.Equal(t, 2, len(logs))

	option.Address = "0xc48e9635c678785927b05eb8ff5c8a8df898dd76c5b91329117ce8cbf429605a"
	filter, err = NewFilter(nil, cfg, &option)

	assert.Nil(t, err)
	logs = filter.filterReceipt(receipt)
	assert.Equal(t, 0, len(logs))
}

func Test_filterEvmTxLogs(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	var evmTx ctypes.EVMTxLogPerBlk
	evmTx.BlockHash = common.FromHex("0xc48e9635c678785927b05eb8ff5c8a8df898dd76c5b91329117ce8cbf429605a")
	evmTx.Height = 123
	evmTx.SeqNum = 123
	evmTx.TxAndLogs = []*ctypes.EVMTxAndLogs{
		{
			Tx: &ctypes.Transaction{
				To: "0xd83b69C56834E85e023B1738E69BFA2F0dd52905",
			},
			LogsPerTx: &ctypes.EVMLogsPerTx{
				Logs: []*ctypes.EVMLog{
					{
						Topic: [][]byte{
							common.FromHex("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
							common.FromHex("0x000000000000000000000000495953a743ef169ec5d4ac7b5f786bf2bd56afd5"),
							common.FromHex("0x0000000000000000000000006eb76c83ad12b865a43cfab57d5f2af1288dd51d"),
						},
						Data: common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"),
					},
					{
						Topic: [][]byte{
							common.FromHex("0xd0943372c08b438a88d4b39d77216901079eda9ca59d45349841c099083b6830"),
							common.FromHex("0x000000000000000000000000495953a743ef169ec5d4ac7b5f786bf2bd56afd5"),
						},
						Data: common.FromHex("0x000000000000000000000000000000000000000000000000000000000000001500000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000002a355000000000000000000000000000000000000000000000000000000000000004d01000000000000000200000000000000001bc16d674ec80000495953a743ef169ec5d4ac7b5f786bf2bd56afd52c22d3715e8e60db6fd784f208ceebedfeac515dcff6ef51b78c80ded64f0e4200000000000000000000000000000000000000"),
					},
				},
			},
		},
	}

	filter, err := NewFilter(nil, cfg, nil)
	assert.Nil(t, err)
	logs := filter.FilterEvmTxLogs(&evmTx)
	assert.Equal(t, 2, len(logs))
	var option FilterQuery
	option.Address = "0xd83b69C56834E85e023B1738E69BFA2F0dd52905"
	option.Topics = append(option.Topics, []interface{}{"0xd0943372c08b438a88d4b39d77216901079eda9ca59d45349841c099083b6830"})
	filter, err = NewFilter(nil, cfg, &option)
	assert.Nil(t, err)
	logs = filter.FilterEvmTxLogs(&evmTx)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "0x"+common.Bytes2Hex(evmTx.GetTxAndLogs()[0].GetLogsPerTx().GetLogs()[1].GetData()), logs[0].Data.String())
	option.Topics = nil
	option.Topics = append(option.Topics, "0x3df38f8a3907335f8612e33723bdab159dbdf0ccc27575215f1865f2cb86b0cd")
	filter, err = NewFilter(nil, cfg, &option)
	logs = filter.FilterEvmTxLogs(&evmTx)
	assert.Equal(t, 0, len(logs))

	option.Topics = nil
	option.Topics = append(option.Topics, []interface{}{"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"})
	filter, err = NewFilter(nil, cfg, &option)
	assert.Nil(t, err)
	logs = filter.FilterEvmTxLogs(&evmTx)
	assert.Equal(t, 1, len(logs))
	option.Address = "0xc48e9635c678785927b05eb8ff5c8a8df898dd76c5b91329117ce8cbf429605a"
	filter, err = NewFilter(nil, cfg, &option)
	assert.Nil(t, err)
	logs = filter.FilterEvmTxLogs(&evmTx)
	assert.Equal(t, 0, len(logs))

	var optionB FilterQuery
	optionB.Address = "0xd83b69C56834E85e023B1738E69BFA2F0dd52905"
	//optionB.Topics = []common.Hash{common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")}
	filter, err = NewFilter(nil, cfg, &optionB)
	assert.Nil(t, err)
	logs = filter.FilterEvmTxLogs(&evmTx)
	assert.Equal(t, 2, len(logs))
}
