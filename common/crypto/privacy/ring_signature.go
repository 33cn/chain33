package privacy

import (
	"unsafe"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	"gitlab.33.cn/chain33/chain33/types"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/common"
)

type PublicKey [32]byte
type SecreKey [32]byte

//type KeyImage [32]byte
type Sign [64]byte

type rs_comm_ab struct {
	a [32]byte
	b [32]byte
}

type rs_comm struct {
	ab []rs_comm_ab
}

func randomScalar(res *[32]byte) {
	var tmp [64]byte
	copy(tmp[:], crypto.CRandBytes(64))
	edwards25519.ScReduce(res, &tmp)
	/*
		for i := 0; i < 32; i++ {
			res[i] = byte(i+7)
		}
	*/
}

func generateKeyImage(pub *PublicKey, sec *SecreKey, image *KeyImage) {
	var point edwards25519.ExtendedGroupElement
	var point2 edwards25519.ProjectiveGroupElement
	p := (*[32]byte)(unsafe.Pointer(sec))
	// Hp(P)
	edwards25519.HashToEc(pub[:], &point)
	//x * Hp(P)
	edwards25519.GeScalarMult(&point2, p, &point)
	point2.ToBytes((*[32]byte)(unsafe.Pointer(image)))
}

func generateRingSignature(data []byte, image *KeyImage, pubs []*PublicKey, sec *SecreKey, signs []*Sign, index int) error {
	var sum, k, h, tmp [32]byte
	var image_pre edwards25519.DsmPreCompGroupElement
	var image_unp edwards25519.ExtendedGroupElement
	var buf []byte
	buf = append(buf, data...)

	if !edwards25519.GeFromBytesVartime(&image_unp, (*[32]byte)(unsafe.Pointer(image))) {
		return errors.New("geFrombytesVartime failed.")
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
				return errors.New("geFrombytesVartime failed.")
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

func checkRingSignature(prefix_hash []byte, image *KeyImage, pubs []*PublicKey, signs []*Sign) bool {
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

// 创建一个环签名
// data 交易哈希值
// utxos 交易的入参
// realUtxoIndex 交易中真正需要签名的交易索引
// sec 私钥
// keyImage 密钥金象
func GenerateRingSignature(data []byte, utxos []*types.UTXO, sec *SecreKey, realUtxoIndex int, keyImage []byte, signs []*Sign) error {
	pubs := make([]*PublicKey, len(utxos))
	for i := 0; i < len(utxos); i++ {
		pub := &PublicKey{}
		copy(pub[:], utxos[i].OnetimePubkey)
		pubs[i] = pub
	}
	h := common.BytesToHash(data)
	var image KeyImage
	copy(image[:], keyImage)
	return generateRingSignature(h.Bytes(), &image, pubs, sec, signs, realUtxoIndex)
}

func GenerateKeyImage(pub *PublicKey, sec *SecreKey, image *KeyImage) {
	generateKeyImage(pub, sec, image)
}

func CheckRingSignature(data []byte, image *KeyImage, pubs []*PublicKey, signs []*Sign) bool {
	h := common.BytesToHash(data)
	return checkRingSignature(h.Bytes(), image, pubs, signs)
}
