package strategy

type simpleCreateExecProjStrategy struct {
	strategyBasic
}

func (this *simpleCreateExecProjStrategy) Run() error {
	mlog.Info("Begin run chain33 simple create dapp project.")
	defer mlog.Info("Run chain33 simple create dapp project finish.")
	return this.runImpl()
}

func (this *simpleCreateExecProjStrategy) runImpl() error {
	return nil
}