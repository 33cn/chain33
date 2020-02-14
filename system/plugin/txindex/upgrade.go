package txindex

import (
	"fmt"

	"github.com/33cn/chain33/system/plugin"
	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

// 这个是value cfg.CalcTxKeyValue(&txresult)

// CalcTxPrefixOld Tx prefix
func CalcTxPrefixOld() []byte {
	return types.TxHashPerfix
}

// CalcTxShortHashPerfixOld short tx
func CalcTxShortHashPerfixOld() []byte {
	return types.TxShortHashPerfix
}

// CalcTxPrefix Calc Tx Prefix
func CalcTxPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, "TX"))
}

// CalcTxShortPerfix Calc Tx Short Prefix
func CalcTxShortPerfix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, "STX"))
}

func (p *txindexPlugin) Upgrade() error {
	toVersion := 2
	elog.Info("Upgrade start", "to_version", toVersion, "plugin", name)
	version, err := plugins.GetVersion(p.GetLocalDB(), name)
	if err != nil {
		return errors.Wrap(err, "Upgrade get version")
	}
	if version >= toVersion {
		elog.Debug("Upgrade not need to upgrade", "current_version", version, "to_version", toVersion)
		return nil
	}
	prefixes := []struct {
		from []byte
		to   []byte
	}{
		{CalcTxPrefixOld(), CalcTxPrefix(name)},
		{CalcTxShortHashPerfixOld(), CalcTxShortPerfix(name)},
	}

	for _, prefix := range prefixes {
		err := plugin.UpgradeOneKey(p.GetLocalDB(), prefix.from, prefix.to)
		if err != nil {
			return err
		}
	}

	err = plugins.SetVersion(p.GetLocalDB(), name, toVersion)
	if err != nil {
		return errors.Wrap(err, "Upgrade setVersion")
	}

	elog.Info("Upgrade upgrade done")
	return nil
}
