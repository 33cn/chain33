package consensus

import "github.com/33cn/chain33/queue"

// Finalizer block finalize
type Finalizer interface {
	Init(base *BaseClient, subCfg []byte)
	ProcessMsg(msg *queue.Message) (processed bool)
}

var finalizers = make(map[string]Finalizer)

// RegFinalizer register committer
func RegFinalizer(name string, f Finalizer) {

	if f == nil {
		panic("RegCommitter: committer is nil")
	}
	if _, dup := committers[name]; dup {
		panic("RegCommitter: duplicate committer " + name)
	}
	finalizers[name] = f
}

// LoadFinalizer load
func LoadFinalizer(name string) Finalizer {

	return finalizers[name]
}
