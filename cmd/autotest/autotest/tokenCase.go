package autotest

type TokenPreCreateCase struct {
	BaseCase
	//From string `toml:"from"`
	//Amount string `toml:"amount"`
}

type TokenPreCreatePack struct {
	BaseCasePack
}

type TokenFinishCreateCase struct {
	BaseCase
	//From string `toml:"from"`
	//Amount string `toml:"amount"`
}

type TokenFinishCreatePack struct {
	BaseCasePack
}

type TokenRevokeCase struct {
	BaseCase
}

type TokenRevokePack struct {
	BaseCasePack
}

func (testCase *TokenPreCreateCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &TokenPreCreatePack{}, packID)
}

func (testCase *TokenRevokeCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &TokenRevokePack{}, packID)
}

func (testCase *TokenFinishCreateCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &TokenFinishCreatePack{}, packID)
}
