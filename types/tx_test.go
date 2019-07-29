// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"

	_ "github.com/33cn/chain33/system/crypto/init"
	"github.com/stretchr/testify/assert"
)

func TestCreateGroupTx(t *testing.T) {
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx2 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630de92c3828ad194b26d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx3 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630b0d6c895c4d28efe5d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	tx21, _ := hex.DecodeString(tx2)
	tx31, _ := hex.DecodeString(tx3)
	var tx12 Transaction
	Decode(tx11, &tx12)
	var tx22 Transaction
	Decode(tx21, &tx22)
	var tx32 Transaction
	Decode(tx31, &tx32)

	group, err := CreateTxGroup([]*Transaction{&tx12, &tx22, &tx32}, GInt("MinFee"))
	if err != nil {
		t.Error(err)
		return
	}
	err = group.Check(0, GInt("MinFee"), GInt("MaxFee"))
	if err != nil {
		for i := 0; i < len(group.Txs); i++ {
			t.Log(group.Txs[i].JSON())
		}
		t.Error(err)
		return
	}
	newtx := group.Tx()
	grouptx := hex.EncodeToString(Encode(newtx))
	t.Log(grouptx)
}

func TestCreateParaGroupTx(t *testing.T) {
	tempTitle := GetTitle()
	SetTitleOnlyForTest("chain33")
	testHeight := int64(1687250 + 1)
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx2 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630de92c3828ad194b26d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx3 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630b0d6c895c4d28efe5d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	tx21, _ := hex.DecodeString(tx2)
	tx31, _ := hex.DecodeString(tx3)
	var tx12 Transaction
	Decode(tx11, &tx12)
	var tx22 Transaction
	Decode(tx21, &tx22)
	var tx32 Transaction
	Decode(tx31, &tx32)

	tx12.Execer = []byte("user.p.test.token")
	tx22.Execer = []byte("token")
	tx32.Execer = []byte("user.p.test.ticket")

	feeRate := GInt("MinFee")
	//SetFork("", "ForkTxGroupPara", 0)
	group, err := CreateTxGroup([]*Transaction{&tx12, &tx22, &tx32}, feeRate)
	if err != nil {
		t.Error(err)
		return
	}
	err = group.Check(testHeight, GInt("MinFee"), GInt("MaxFee"))
	if err != nil {
		for i := 0; i < len(group.Txs); i++ {
			t.Log(group.Txs[i].JSON())
		}
		//t.Error(err)

	}
	assert.Equal(t, ErrTxGroupParaMainMixed, err)

	tx22.Execer = []byte("user.p.para.token")
	group, err = CreateTxGroup([]*Transaction{&tx12, &tx22, &tx32}, feeRate)
	if err != nil {
		t.Error(err)
		return
	}
	err = group.Check(testHeight, GInt("MinFee"), GInt("MaxFee"))
	assert.Equal(t, ErrTxGroupParaCount, err)

	tx22.Execer = []byte("user.p.test.paracross")
	group, err = CreateTxGroup([]*Transaction{&tx12, &tx22, &tx32}, feeRate)
	if err != nil {
		t.Error(err)
		return
	}
	err = group.Check(testHeight, GInt("MinFee"), GInt("MaxFee"))
	assert.Nil(t, err)
	newtx := group.Tx()
	grouptx := hex.EncodeToString(Encode(newtx))
	t.Log(grouptx)
	SetTitleOnlyForTest(tempTitle)
}

func TestCreateGroupTxWithSize(t *testing.T) {
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx2 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630de92c3828ad194b26d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx3 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630b0d6c895c4d28efe5d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	tx21, _ := hex.DecodeString(tx2)
	tx31, _ := hex.DecodeString(tx3)

	len150str := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b5366761122"
	len130str := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c76364339715734617678733245375"
	len105str := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7633"

	var tx12 Transaction
	Decode(tx11, &tx12)
	tx12.Fee = 1
	//构造临界size， fee=1,构建时候计算出size是998， 构建之前fee是100000，构建之后tx[0].fee=200000，原代码会出错
	extSize := []byte(len150str + len150str + len150str + len105str)
	tx12.Payload = append(tx12.Payload, extSize...)

	var tx22 Transaction
	Decode(tx21, &tx22)
	//构造临界size， 有没有header的场景
	extSize = []byte(len150str + len150str + len150str + len130str)
	tx22.Payload = append(tx22.Payload, extSize...)
	var tx32 Transaction
	Decode(tx31, &tx32)

	group, err := CreateTxGroup([]*Transaction{&tx12, &tx22, &tx32}, GInt("MinFee"))
	if err != nil {
		t.Error(err)
		return
	}

	err = group.Check(0, GInt("MinFee"), GInt("MaxFee"))
	if err != nil {
		for i := 0; i < len(group.Txs); i++ {
			t.Log(group.Txs[i].JSON())
		}
		t.Error(err)
		return
	}
	newtx := group.Tx()
	grouptx := hex.EncodeToString(Encode(newtx))
	t.Log(grouptx)
}

func TestDecodeTx(t *testing.T) {
	signtx := "0a05636f696e73120e18010a0a1080c2d72f1a036f74651a6d0801122102504fa1c28caaf1d5a20fefb87c50a49724ff401043420cb3ba271997eb5a43871a46304402200e566613679e8fe645990adb8ed6aa8c46060d944f5bab358e2c78443c3eed53022049d671e596d48f091dae3558b6fd811250412101765ba0dec5cc4188a180088720e0a71230f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034ada050afe010a05636f696e73120e18010a0a1080c2d72f1a036f74651a6d0801122102504fa1c28caaf1d5a20fefb87c50a49724ff401043420cb3ba271997eb5a43871a46304402200e566613679e8fe645990adb8ed6aa8c46060d944f5bab358e2c78443c3eed53022049d671e596d48f091dae3558b6fd811250412101765ba0dec5cc4188a180088720e0a71230f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a20a9ef5454033a9ab080360470291e9b1f63881ff9a03c3c09a06e7200688d019852209a56c5dbff8b246e32d3ad0534f42dbbae88cf4b0bed24fe6420e06d59187c690afb010a05636f696e73120e18010a0a1080c2d72f1a036f74651a6e0801122102504fa1c28caaf1d5a20fefb87c50a49724ff401043420cb3ba271997eb5a43871a473045022100ac7acba851854179f0d574428e8c5a4c69d4431604e8626fd7ace87a8abe1f6c022039eb3f7ec190030b2c7e32457972482b3d074521856ea0d09820071b39ffb4b930de92c3828ad194b26d3a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a20a9ef5454033a9ab080360470291e9b1f63881ff9a03c3c09a06e7200688d0198522036bb9ca17aeef20b6e9afcd5ba52d89f5109db5b5d2aee200723b2bb0c7e9aa30ad8010a05636f696e73120e18010a0a1080c2d72f1a036f74651a6d0801122102504fa1c28caaf1d5a20fefb87c50a49724ff401043420cb3ba271997eb5a43871a4630440220094e12621f235ea46e99d21f30e8be510a52e6d92410b35e307936ce61aafe9602207fcdeb51825af222159c82b74ab2386d01263e4903d4a0cf96426c1b48bd083130b0d6c895c4d28efe5d3a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a20a9ef5454033a9ab080360470291e9b1f63881ff9a03c3c09a06e7200688d019852209a56c5dbff8b246e32d3ad0534f42dbbae88cf4b0bed24fe6420e06d59187c69"
	var tx Transaction
	txhex, _ := hex.DecodeString(signtx)
	Decode(txhex, &tx)
	group, err := tx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	if !group.CheckSign() {
		t.Error("group: sign should be no err")
	}
	//txs[0] 的hash 应该和 整体的hash相同
	if string(tx.Hash()) != string(group.Txs[0].Hash()) {
		t.Error("group: tx.Hash ==  group.Txs[0].Hash()")
	}
	for i := 0; i < len(group.Txs); i++ {
		//t.Log(group.Txs[i].Json())
		if group.Txs[i].IsExpire(10, Now().Unix()) {
			t.Error("group txs[i]: Expire not set so, no exprie forever")
		}
		if !group.Txs[i].CheckSign() {
			t.Error("group txs[i]: sign should be no err")
		}
	}
	if group.IsExpire(10, Now().Unix()) {
		t.Error("group: Expire not set so, no exprie forever")
	}
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(GetSignName("", SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func TestSignGroupTx(t *testing.T) {
	var err error
	privkey := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	unsignTx := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520e0a71230f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a8b030a8f010a05636f696e73120e18010a0a1080c2d72f1a036f746520e0a71230f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a20a9ef5454033a9ab080360470291e9b1f63881ff9a03c3c09a06e7200688d019852209a56c5dbff8b246e32d3ad0534f42dbbae88cf4b0bed24fe6420e06d59187c690a8b010a05636f696e73120e18010a0a1080c2d72f1a036f746530de92c3828ad194b26d3a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a20a9ef5454033a9ab080360470291e9b1f63881ff9a03c3c09a06e7200688d0198522036bb9ca17aeef20b6e9afcd5ba52d89f5109db5b5d2aee200723b2bb0c7e9aa30a690a05636f696e73120e18010a0a1080c2d72f1a036f746530b0d6c895c4d28efe5d3a22313271796f6361794e46374c7636433971573461767873324537553431664b53667640034a20a9ef5454033a9ab080360470291e9b1f63881ff9a03c3c09a06e7200688d019852209a56c5dbff8b246e32d3ad0534f42dbbae88cf4b0bed24fe6420e06d59187c69"
	var tx Transaction
	txhex, _ := hex.DecodeString(unsignTx)
	Decode(txhex, &tx)
	group, err := tx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	if group == nil {
		t.Errorf("signN sign a not group tx")
		return
	}
	for i := 0; i < len(group.GetTxs()); i++ {
		err = group.SignN(i, SECP256K1, privkey)
		if err != nil {
			t.Error(err)
			return
		}
	}
	err = group.Check(0, GInt("MinFee"), GInt("MaxFee"))
	if err != nil {
		t.Error(err)
		return
	}
	newtx := group.Tx()
	signedtx := hex.EncodeToString(Encode(newtx))
	t.Log(signedtx)
}

func BenchmarkTxHash(b *testing.B) {
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 Transaction
	Decode(tx11, &tx12)
	for i := 0; i < b.N; i++ {
		tx12.Hash()
	}
}

func TestParseExpire(t *testing.T) {
	expire := ""
	_, err := ParseExpire(expire)
	assert.Equal(t, ErrInvalidParam, err)

	expire = "H:123"
	exp, _ := ParseExpire(expire)
	assert.Equal(t, int64(4611686018427388027), exp)

	expire = "H:-2"
	_, err = ParseExpire(expire)
	assert.Equal(t, ErrHeightLessZero, err)

	expire = "123"
	exp, err = ParseExpire(expire)
	assert.Nil(t, err)
	assert.Equal(t, int64(123), exp)

	expire = "123s"
	exp, err = ParseExpire(expire)
	assert.Nil(t, err)
	assert.Equal(t, int64(123000000000), exp)

}

func BenchmarkHash(b *testing.B) {
	tx := &Transaction{Payload: []byte("xxxxxxxxxxxxdggrgrgrgrgrgrgrrhthththhth"), Execer: []byte("hello")}
	for i := 0; i < b.N; i++ {
		tx.Hash()
	}
}

func TestSetGroupExpire(t *testing.T) {
	rawtx := "0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720c0843d30aab4d59684b5cce7143a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4ab50c0aa3010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720c0843d30aab4d59684b5cce7143a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522008217c413b035fddd8f34a303e90a29e661746ed9b23a97768c1f25817c2c3450a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a673094fbcabe96c99ea7163a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f552203c6a2b11cce466891f084b49450472b1d4c39213f63117d3d4ce2a3851304ebc0a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730c187fb80fe88ce9e3c3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522066419d70492f757d7285fd226dff62da8d803c8121ded95242d222dbb10f2d9b0a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a673098aa929ab292b3f0023a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f552202bab08051d24fe923f66c8aeea4ce3f425d47a72f7c5c230a2b1427e04e2eb510a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730bfe9abb3edc6d9cb163a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f55220e1ba0493aa431ea3071026bd8dfa8280efab53ce86441fc474a1c19550a554ba0a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730d2e196a8ecada9d53e3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522016600fbfa23b3f0e8f9a14b716ce8f4064c091fbf6fa94489bc9d14b5b6049a60a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730a0b7b1b1dda2f4c5743a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522089d0442d76713369022499d054db65ccacbf5c627a525bd5454e0a30d23fa2990a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730c5838f94e2f49acb4b3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522018f208938606b390d752898332a84a9fbb900c2ed55ec33cd54d09b1970043b90a9f010a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a67308dfddb82faf7dfc4113a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522013002bab7a9c65881bd937a6fded4c3959bb631fa84434572970c1ec3e6fccf90a7d0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6730b8b082d799a4ddc93a3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f400a4a201f533ac07c3fc4c716f65cdb0f1f02e7f5371b5164277210dafb1dbdd4a5f4f5522008217c413b035fddd8f34a303e90a29e661746ed9b23a97768c1f25817c2c345"
	var tx Transaction
	txhex, _ := hex.DecodeString(rawtx)
	Decode(txhex, &tx)
	group, err := tx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	for _, tmptx := range group.GetTxs() {
		if tmptx.GetExpire() != 0 {
			t.Error("TestSetGroupExpire Expire !=0", "tx", tmptx)
		}
	}

	//设置交易组过期时间
	for i := 0; i < len(group.Txs); i++ {
		group.SetExpire(i, time.Duration(120))
	}
	group.RebuiltGroup()

	//校验重组后的交易组
	firsttxhash := group.GetTxs()[0].Hash()
	for _, tmptx := range group.GetTxs() {
		if string(tmptx.GetHeader()) != string(firsttxhash) {
			t.Error("TestSetGroupExpire group: tx.Hash !=  group.Txs[0].Hash()")
		}
	}

	for _, tmptx := range group.GetTxs() {
		if tmptx.GetExpire() == 0 {
			t.Error("TestSetGroupExpire Expire == 0", "tx", tmptx)
		}
	}
}
