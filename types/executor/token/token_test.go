package token

import (
	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
	"testing"
)

func init() {
	Init()
}

func TestCreateRawTokenPreCreateTx(t *testing.T) {
	//预创建主链上的token
	tokenPreCreate := &TokenPreCreateTx{
		0,
		"RNG",
		"RNG",
		"fzmtest",
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		5 * 1e16,
		1e5,
		"",
	}
	preCreateTx, err := CreateRawTokenPreCreateTx(tokenPreCreate)
	assert.NotNil(t, preCreateTx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("token"), preCreateTx.Execer)
	assert.Equal(t, tokenPreCreate.Fee, preCreateTx.Fee)
	assert.Equal(t, address.ExecAddress("token"), preCreateTx.To)
	//预创建平行链上的token
	tokenPreCreate = &TokenPreCreateTx{
		0,
		"RNG",
		"RNG",
		"fzmtest",
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		5 * 1e16,
		1e5,
		"user.p.fzmtest.",
	}
	preCreateTx, err = CreateRawTokenPreCreateTx(tokenPreCreate)
	assert.NotNil(t, preCreateTx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("user.p.fzmtest.token"), preCreateTx.Execer)
	assert.Equal(t, tokenPreCreate.Fee, preCreateTx.Fee)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), preCreateTx.To)
}

func TestCreateRawTokenFinishTx(t *testing.T) {
	tokenFinsh := &TokenFinishTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		"RNG",
		1e5,
		"",
	}
	tokenFinshTx, err := CreateRawTokenFinishTx(tokenFinsh)
	assert.NotNil(t, tokenFinshTx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("token"), tokenFinshTx.Execer)
	assert.Equal(t, tokenFinsh.Fee, tokenFinshTx.Fee)
	assert.Equal(t, address.ExecAddress("token"), tokenFinshTx.To)

	tokenFinsh = &TokenFinishTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		"RNG",
		1e5,
		"user.p.fzmtest.",
	}
	tokenFinshTx, err = CreateRawTokenFinishTx(tokenFinsh)
	assert.NotNil(t, tokenFinshTx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("user.p.fzmtest.token"), tokenFinshTx.Execer)
	assert.Equal(t, tokenFinsh.Fee, tokenFinshTx.Fee)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), tokenFinshTx.To)
}

func TestCreateRawTokenRevokeTx(t *testing.T) {
	//在主链上
	tokenRevoke := &TokenRevokeTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		"RNG",
		1e5,
		"",
	}
	tokenRevokeTx, err := CreateRawTokenRevokeTx(tokenRevoke)
	assert.NotNil(t, tokenRevokeTx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("token"), tokenRevokeTx.Execer)
	assert.Equal(t, tokenRevoke.Fee, tokenRevokeTx.Fee)
	assert.Equal(t, address.ExecAddress("token"), tokenRevokeTx.To)
	//加上paraName,在平行链上
	tokenRevoke = &TokenRevokeTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		"RNG",
		1e5,
		"user.p.fzmtest.",
	}
	tokenRevokeTx, err = CreateRawTokenRevokeTx(tokenRevoke)
	assert.NotNil(t, tokenRevokeTx)
	assert.Nil(t, err)
	assert.Equal(t, []byte("user.p.fzmtest.token"), tokenRevokeTx.Execer)
	assert.Equal(t, tokenRevoke.Fee, tokenRevokeTx.Fee)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), tokenRevokeTx.To)
}

func TestCreateTokenTransfer(t *testing.T) {
	//平行链上，传入完整的execName
	types.SetTitle("user.p.fzmtest.")
	createTx := &types.CreateTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		10000,
		1e5,
		"this is test",
		false,
		true,
		"RNG",
		"user.p.fzmtest.token",
	}
	tx := CreateTokenTransfer(createTx)
	assert.NotNil(t, tx)
	assert.Equal(t, []byte("user.p.fzmtest.token"), tx.Execer)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), tx.To)
	//传入,execName=token
	createTx = &types.CreateTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		10000,
		1e5,
		"this is test",
		false,
		true,
		"RNG",
		"token",
	}
	tx = CreateTokenTransfer(createTx)
	assert.NotNil(t, tx)
	assert.Equal(t, []byte("user.p.fzmtest.token"), tx.Execer)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), tx.To)
	//execName 传空字符串
	createTx = &types.CreateTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		10000,
		1e5,
		"this is test",
		false,
		true,
		"RNG",
		"",
	}
	tx = CreateTokenTransfer(createTx)
	assert.NotNil(t, tx)
	assert.Equal(t, []byte("user.p.fzmtest.token"), tx.Execer)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), tx.To)

	//在主链上
	types.SetTitle("chain33")
	//在主链上构造平行链交易，此时必须要传入完整的execName,不能省略
	createTx = &types.CreateTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		10000,
		1e5,
		"this is test",
		false,
		true,
		"RNG",
		"user.p.fzmtest.token",
	}
	tx = CreateTokenTransfer(createTx)
	assert.NotNil(t, tx)
	assert.Equal(t, []byte("user.p.fzmtest.token"), tx.Execer)
	assert.Equal(t, address.ExecAddress("user.p.fzmtest.token"), tx.To)
	//execName传入token,或者空字符串，默认是构建主链自己的交易
	createTx = &types.CreateTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		10000,
		1e5,
		"this is test",
		false,
		true,
		"RNG",
		"token",
	}
	tx = CreateTokenTransfer(createTx)
	assert.NotNil(t, tx)
	assert.Equal(t, []byte("token"), tx.Execer)
	assert.Equal(t, createTx.To, tx.To)
	//execName 传空字符串
	createTx = &types.CreateTx{
		"16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh",
		10000,
		1e5,
		"this is test",
		false,
		true,
		"RNG",
		"",
	}
	tx = CreateTokenTransfer(createTx)
	assert.NotNil(t, tx)
	assert.Equal(t, []byte("token"), tx.Execer)
	assert.Equal(t, createTx.To, tx.To)
}
