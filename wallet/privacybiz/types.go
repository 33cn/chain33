package privacybiz

import "gitlab.33.cn/chain33/chain33/common/crypto/privacy"

type addrAndprivacy struct {
	PrivacyKeyPair *privacy.Privacy
	Addr           *string
}
