package tests

type VMCase struct {
	name string
	env  EnvJson
	exec ExecJson
	gas  int64
	logs string
	out  string
	err  string
	pre  map[string]AccountJson
	post map[string]AccountJson
}

type EnvJson struct {
	currentCoinbase   string
	currentDifficulty int64
	currentGasLimit   int64
	currentNumber     int64
	currentTimestamp  int64
}

type ExecJson struct {
	address  string
	caller   string
	code     string
	data     string
	gas      int64
	gasPrice int64
	origin   string
	value    int64
}

type AccountJson struct {
	balance int64
	code    string
	nonce   int64
	storage map[string]string
}
