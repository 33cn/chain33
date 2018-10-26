package privacy

import (
	"unsafe"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type Sign [64]byte

func randomScalar(res *[32]byte) {
	var tmp [64]byte
	copy(tmp[:], crypto.CRandBytes(64))
	edwards25519.ScReduce(res, &tmp)
}

func generateKeyImage(pub *PubKeyPrivacy, sec *PrivKeyPrivacy, image *KeyImage) error {
	var point edwards25519.ExtendedGroupElement
	var point2 edwards25519.ProjectiveGroupElement
	if pub == nil || sec == nil || image == nil {
		return types.ErrInvalidParam
	}
	p := (*[32]byte)(unsafe.Pointer(sec))
	// Hp(P)
	edwards25519.HashToEc(pub[:], &point)
	//x * Hp(P)
	edwards25519.GeScalarMult(&point2, p, &point)
	point2.ToBytes((*[32]byte)(unsafe.Pointer(image)))
	return nil
}

func generateRingSignature(data []byte, image *KeyImage, pubs []*PubKeyPrivacy, sec *PrivKeyPrivacy, signs []*Sign, index int) error {
	var sum, k, h, tmp [32]byte
	var image_pre edwards25519.DsmPreCompGroupElement
	var image_unp edwards25519.ExtendedGroupElement
	var buf []byte
	buf = append(buf, data...)

	if !edwards25519.GeFromBytesVartime(&image_unp, (*[32]byte)(unsafe.Pointer(image))) {
		privacylog.Error("generateRingSignature", "from image failed.")
		return privacytypes.ErrGeFromBytesVartime
	}
	edwards25519.GeDsmPrecomp(&image_pre, &image_unp)
	for i := 0; i < len(pubs); i++ {
		var tmp2 edwards25519.ProjectiveGroupElement
		var tmp3 edwards25519.ExtendedGroupElement
		pubkey := pubs[i]
		sign := signs[i]
		pa := (*[32]byte)(unsafe.Pointer(sign))
		pb := (*[32]byte)(unsafe.Pointer(&sign[32]))
		if i == index {
			// in case: i == index
			// generate q_i
			randomScalar(&k)
			// q_i*G
			edwards25519.GeScalarMultBase(&tmp3, &k)
			//save q_i*Gp
			tmp3.ToBytes(&tmp)
			buf = append(buf, tmp[:]...)
			// Hp(Pi)
			edwards25519.HashToEc(pubkey[:], &tmp3)
			// q_i*Hp(Pi)
			edwards25519.GeScalarMult(&tmp2, &k, &tmp3)
			// save q_i*Hp(Pi)
			tmp2.ToBytes(&tmp)
			buf = append(buf, tmp[:]...)
		} else {
			// in case: i != realUtxoIndex
			randomScalar(pa)
			randomScalar(pb)
			if !edwards25519.GeFromBytesVartime(&tmp3, (*[32]byte)(unsafe.Pointer(pubkey))) {
				return privacytypes.ErrGeFromBytesVartime
			}
			// (r, a, A, b)
			// r = a  * A   + b   * G
			//    Wi * Pi + q_i  * G
			// (r, Wi, Pi, q_i)
			edwards25519.GeDoubleScalarMultVartime(&tmp2, pa, &tmp3, pb)
			// save q_i*G + Wi*Pi
			tmp2.ToBytes(&tmp)
			buf = append(buf, tmp[:]...)
			// Hp(Pi)
			edwards25519.HashToEc(pubkey[:], &tmp3)
			// q_i*Hp(Pi) + Wi*I
			// (r, a, A, b, B)
			// r = a  * A   + b   * B
			//    Wi * Hp(Pi) + q_i  * I
			// (r, Wi, Hp(Pi), q_i, I)
			edwards25519.GeDoubleScalarmultPrecompVartime(&tmp2, pb, &tmp3, pa, &image_pre)
			// save q_i*Hp(Pi) + Wi*I
			tmp2.ToBytes(&tmp)
			buf = append(buf, tmp[:]...)
			// sum_c = sum(c_0,...c_n)
			edwards25519.ScAdd(&sum, &sum, (*[32]byte)(unsafe.Pointer(sign)))
		}
	}
	// c = Hs(m; L1... Ln; R1... Rn)
	hash2scalar(buf, &h)
	sign := signs[index]
	c := (*[32]byte)(unsafe.Pointer(sign))
	s := (*[32]byte)(unsafe.Pointer(&sign[32]))
	// c_s = c - Sum(c_0, c_0,...c_n)
	edwards25519.ScSub(c, &h, &sum)
	// r_s = q_s - c_s*x
	// (s, a, b, c)
	// s = c - a*b
	edwards25519.ScMulSub(s, c, (*[32]byte)(unsafe.Pointer(sec)), &k)
	return nil
}

func checkRingSignature(prefix_hash []byte, image *KeyImage, pubs []*PubKeyPrivacy, signs []*Sign) bool {
	var sum, h, tmp [32]byte
	var image_unp edwards25519.ExtendedGroupElement
	var image_pre edwards25519.DsmPreCompGroupElement
	var buf []byte
	buf = append(buf, prefix_hash...)

	if !edwards25519.GeFromBytesVartime(&image_unp, (*[32]byte)(unsafe.Pointer(image))) {
		return false
	}
	edwards25519.GeDsmPrecomp(&image_pre, &image_unp)
	for i := 0; i < len(pubs); i++ {
		var tmp2 edwards25519.ProjectiveGroupElement
		var tmp3 edwards25519.ExtendedGroupElement
		pub := pubs[i]
		sign := signs[i]
		pa := (*[32]byte)(unsafe.Pointer(sign))
		pb := (*[32]byte)(unsafe.Pointer(&sign[32]))
		if !edwards25519.ScCheck(pa) || !edwards25519.ScCheck(pb) {
			return false
		}
		if !edwards25519.GeFromBytesVartime(&tmp3, (*[32]byte)(unsafe.Pointer(pub))) {
			return false
		}
		//L'_i = r_i * G + c_i * Pi
		edwards25519.GeDoubleScalarMultVartime(&tmp2, pa, &tmp3, pb)
		//save: L'_i = r_i * G + c_i * Pi
		tmp2.ToBytes(&tmp)
		buf = append(buf, tmp[:]...)
		//Hp(Pi)
		edwards25519.HashToEc(pub[:], &tmp3)
		//R'_i = r_i * Hp(Pi) + c_i * I
		edwards25519.GeDoubleScalarmultPrecompVartime(&tmp2, pb, &tmp3, pa, &image_pre)
		//save: R'_i = r_i * Hp(Pi) + c_i * I
		tmp2.ToBytes(&tmp)
		buf = append(buf, tmp[:]...)
		//sum_c = sum(c_0,...c_n)
		edwards25519.ScAdd(&sum, &sum, (*[32]byte)(unsafe.Pointer(pa)))
	}
	//Hs(m, L'_0...L'_n, R'_0...R'_n)
	hash2scalar(buf, &h)
	//sum_c ?== Hs(m, L'_0...L'_n, R'_0...R'_n)
	edwards25519.ScSub(&h, &h, &sum)
	return edwards25519.ScIsNonZero(&h) == 0
}

func GenerateRingSignature(datahash []byte, utxos []*privacytypes.UTXOBasic, privKey []byte, realUtxoIndex int, keyImage []byte) (*types.RingSignatureItem, error) {
	count := len(utxos)
	signs := make([]*Sign, count)
	pubs := make([]*PubKeyPrivacy, count)

	data := types.RingSignatureItem{}
	data.Signature = make([][]byte, count)
	data.Pubkey = make([][]byte, count)
	for i := 0; i < count; i++ {
		utxo := utxos[i]
		pub := &PubKeyPrivacy{}
		copy(pub[:], utxo.OnetimePubkey)
		pubs[i] = pub
		signs[i] = &Sign{}
		data.Pubkey[i] = append(data.Pubkey[i], pub[:]...)
	}
	var image KeyImage
	copy(image[:], keyImage)
	var sec PrivKeyPrivacy
	copy(sec[:], privKey)
	err := generateRingSignature(datahash, &image, pubs, &sec, signs, realUtxoIndex)
	if err != nil {
		return nil, err
	}
	for i, v := range signs {
		data.Signature[i] = append(data.Signature[i], v[:]...)
	}
	return &data, nil
}

// GenerateKeyImage 根据给定的公钥和私钥信息生成对应的秘钥镜像
func GenerateKeyImage(privkey crypto.PrivKey, pubkey []byte) (*KeyImage, error) {
	var image KeyImage
	var pub PubKeyPrivacy
	var sec PrivKeyPrivacy
	copy(pub[:], pubkey)
	copy(sec[:], privkey.Bytes())
	err := generateKeyImage(&pub, &sec, &image)
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// CheckRingSignature 效验环签名的签名信息
// 效验数据datahash是一个哈
func CheckRingSignature(datahash []byte, signatures *types.RingSignatureItem, publickeys [][]byte, keyimage []byte) bool {
	var image KeyImage
	var pubs []*PubKeyPrivacy
	var signs []*Sign

	if signatures == nil || len(signatures.GetSignature()) != len(publickeys) {
		return false
	}
	// 转换协议
	copy(image[:], keyimage)
	count := len(publickeys)
	pubs = make([]*PubKeyPrivacy, count)
	signs = make([]*Sign, count)
	for i := 0; i < len(publickeys); i++ {
		pub := PubKeyPrivacy{}
		sign := Sign{}
		copy(pub[:], publickeys[i])
		copy(sign[:], signatures.GetSignature()[i])
		pubs[i] = &pub
		signs[i] = &sign
	}

	return checkRingSignature(datahash, &image, pubs, signs)
}
