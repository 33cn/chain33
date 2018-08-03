package privacybiz

import (
	"gitlab.33.cn/chain33/chain33/wallet/bizpolicy"
)

func New() bizpolicy.WalletBizPolicy {
	return &walletPrivacyBiz{}
}
