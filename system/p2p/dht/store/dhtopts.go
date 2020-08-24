package store

import (
	dbm "github.com/33cn/chain33/common/db"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
)

//NewDataStore new...
func NewDataStore(cfg *p2pty.P2PSubConfig) *Persistent {
	var DHTDataDriver string
	var DHTDataPath string
	var DHTDataCache int32

	if cfg != nil {
		DHTDataDriver = cfg.DHTDataDriver
		DHTDataPath = cfg.DHTDataPath
		DHTDataCache = cfg.DHTDataCache
	}

	if DHTDataDriver == "" {
		DHTDataDriver = DefaultDataDriver
	}
	if DHTDataPath == "" {
		DHTDataPath = DefaultDataPath
	}

	if DHTDataCache <= 0 {
		DHTDataCache = DefaultDataCache
	}

	db := dbm.NewDB(DBName, DHTDataDriver, DHTDataPath, DHTDataCache)
	return &Persistent{db}
}

func dataStoreOption(cfg *p2pty.P2PSubConfig) opts.Option {
	return opts.Datastore(NewDataStore(cfg))
}

func validatorOption() opts.Option {
	return opts.NamespacedValidator(DhtStoreNamespace, &validator{})
}
