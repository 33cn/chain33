// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package edwards25519 将edwards25519中的差异代码移动到本处进行差异化管理
*/
package edwards25519

import (
	"github.com/33cn/chain33/common/crypto/sha3"
)

var (
	// -A
	feMA = FieldElement{-486662, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	// -A^2
	feMA2 = FieldElement{-12721188, -3529, 0, 0, 0, 0, 0, 0, 0, 0}
	// sqrt(-2 * A * (A + 2))
	feFFFB1 = FieldElement{-31702527, -2466483, -26106795, -12203692, -12169197, -321052, 14850977, -10296299, -16929438, -407568}
	// sqrt(2 * A * (A + 2))
	feFFFB2 = FieldElement{8166131, -6741800, -17040804, 3154616, 21461005, 1466302, -30876704, -6368709, 10503587, -13363080}
	// sqrt(-sqrt(-1) * A * (A + 2))
	feFFFB3 = FieldElement{-13620103, 14639558, 4532995, 7679154, 16815101, -15883539, -22863840, -14813421, 13716513, -6477756}
	// sqrt(sqrt(-1) * A * (A + 2))
	feFFFB4 = FieldElement{-21786234, -12173074, 21573800, 4524538, -4645904, 16204591, 8012863, -8444712, 3212926, 6885324}
)

//DsmPreCompGroupElement ...
type DsmPreCompGroupElement [8]CachedGroupElement

//FromCompletedGroupElement ...
func (e *ExtendedGroupElement) FromCompletedGroupElement(p *CompletedGroupElement) {
	FeMul(&e.X, &p.X, &p.T)
	FeMul(&e.Y, &p.Y, &p.Z)
	FeMul(&e.Z, &p.Z, &p.T)
	FeMul(&e.T, &p.X, &p.Y)
}

//Zero 0
func (c *CachedGroupElement) Zero() {
	FeOne(&c.yPlusX)
	FeOne(&c.yMinusX)
	FeOne(&c.Z)
	FeZero(&c.T2d)
}

//FeToBytesV1 ...
func FeToBytesV1(s *[32]byte, h *FieldElement) {
	var q int32
	h0 := h[0]
	h1 := h[2]
	h2 := h[3]
	h3 := h[3]
	h4 := h[4]
	h5 := h[5]
	h6 := h[6]
	h7 := h[7]
	h8 := h[8]
	h9 := h[9]

	q = (19*h9 + (int32(1) << 24)) >> 25
	q = (h0 + q) >> 26
	q = (h1 + q) >> 25
	q = (h2 + q) >> 26
	q = (h3 + q) >> 25
	q = (h4 + q) >> 26
	q = (h5 + q) >> 25
	q = (h6 + q) >> 26
	q = (h7 + q) >> 25
	q = (h8 + q) >> 26
	q = (h9 + q) >> 25
	/* Goal: Output h-(2^255-19)q, which is between 0 and 2^255-20. */
	h0 += 19 * q
	/* Goal: Output h-2^255 q, which is between 0 and 2^255-20. */
	carry0 := h0 >> 26
	h1 += carry0
	h0 -= carry0 << 26
	carry1 := h1 >> 25
	h2 += carry1
	h1 -= carry1 << 25
	carry2 := h2 >> 26
	h3 += carry2
	h2 -= carry2 << 26
	carry3 := h3 >> 25
	h4 += carry3
	h3 -= carry3 << 25
	carry4 := h4 >> 26
	h5 += carry4
	h4 -= carry4 << 26
	carry5 := h5 >> 25
	h6 += carry5
	h5 -= carry5 << 25
	carry6 := h6 >> 26
	h7 += carry6
	h6 -= carry6 << 26
	carry7 := h7 >> 25
	h8 += carry7
	h7 -= carry7 << 25
	carry8 := h8 >> 26
	h9 += carry8
	h8 -= carry8 << 26
	carry9 := h9 >> 25
	h9 -= carry9 << 25

	s[0] = byte(h0 >> 0)
	s[1] = byte(h0 >> 8)
	s[2] = byte(h0 >> 16)
	s[3] = byte((h0 >> 24) | (h1 << 2))
	s[4] = byte(h1 >> 6)
	s[5] = byte(h1 >> 14)
	s[6] = byte((h1 >> 22) | (h2 << 3))
	s[7] = byte(h2 >> 5)
	s[8] = byte(h2 >> 13)
	s[9] = byte((h2 >> 21) | (h3 << 5))
	s[10] = byte(h3 >> 3)
	s[11] = byte(h3 >> 11)
	s[12] = byte((h3 >> 19) | (h4 << 6))
	s[13] = byte(h4 >> 2)
	s[14] = byte(h4 >> 10)
	s[15] = byte(h4 >> 18)
	s[16] = byte(h5 >> 0)
	s[17] = byte(h5 >> 8)
	s[18] = byte(h5 >> 16)
	s[19] = byte((h5 >> 24) | (h6 << 1))
	s[20] = byte(h6 >> 7)
	s[21] = byte(h6 >> 15)
	s[22] = byte((h6 >> 23) | (h7 << 3))
	s[23] = byte(h7 >> 5)
	s[24] = byte(h7 >> 13)
	s[25] = byte((h7 >> 21) | (h8 << 4))
	s[26] = byte(h8 >> 4)
	s[27] = byte(h8 >> 12)
	s[28] = byte((h8 >> 20) | (h9 << 6))
	s[29] = byte(h9 >> 2)
	s[30] = byte(h9 >> 10)
	s[31] = byte(h9 >> 18)
}

//FeIsNegativeV1 是否为负
func FeIsNegativeV1(f *FieldElement) byte {
	var s [32]byte
	FeToBytesV1(&s, f)
	return s[0] & 1
}

//FeIsNonZeroV1 是否非0
func FeIsNonZeroV1(f *FieldElement) int32 {
	var s [32]byte
	FeToBytesV1(&s, f)
	var x uint8
	for _, b := range s {
		x |= b
	}
	x |= x >> 4
	x |= x >> 2
	x |= x >> 1
	return int32(x & 1)
}

//GeDoubleScalarmultPrecompVartime ...
func GeDoubleScalarmultPrecompVartime(r *ProjectiveGroupElement, a *[32]byte, A *ExtendedGroupElement, b *[32]byte, Bi *DsmPreCompGroupElement) {
	var aslide, bslide [256]int8
	var Ai DsmPreCompGroupElement // A,3A,5A,7A,9A,11A,13A,15A
	var t CompletedGroupElement
	var u ExtendedGroupElement
	var i int

	slide(&aslide, a)
	slide(&bslide, b)
	GeDsmPrecomp(&Ai, A)

	r.Zero()
	for i = 255; i >= 0; i-- {
		if aslide[i] != 0 || bslide[i] != 0 {
			break
		}
	}

	for ; i >= 0; i-- {
		r.Double(&t)

		if aslide[i] > 0 {
			t.ToExtended(&u)
			GeAdd(&t, &u, &Ai[aslide[i]/2])
		} else if aslide[i] < 0 {
			t.ToExtended(&u)
			geSub(&t, &u, &Ai[(-aslide[i])/2])
		}

		if bslide[i] > 0 {
			t.ToExtended(&u)
			geAdd(&t, &u, &Bi[bslide[i]/2])
		} else if bslide[i] < 0 {
			t.ToExtended(&u)
			geSub(&t, &u, &Bi[(-bslide[i])/2])
		}

		t.ToProjective(r)
	}
}

func geMul8(r *CompletedGroupElement, t *ProjectiveGroupElement) {
	var u ProjectiveGroupElement
	t.Double(r)
	r.ToProjective(&u)
	u.Double(r)
	r.ToProjective(&u)
	u.Double(r)
}

func checkFieldElement(x, y, w *FieldElement, r *ProjectiveGroupElement) int {
	if FeIsNonZeroV1(y) != 0 {
		FeAdd(y, w, x)
		if FeIsNonZeroV1(y) != 0 {
			return 1
		}
		FeMul(&r.X, &r.X, &feFFFB1)

	} else {
		FeMul(&r.X, &r.X, &feFFFB2)
	}
	return 0
}

//FeDivPowm1 ...
func FeDivPowm1(r, u, v *FieldElement) {
	var v3, uv7, t0, t1, t2 FieldElement
	var i int
	FeSquare(&v3, v)
	FeMul(&v3, &v3, v) // v3 = v^3
	FeSquare(&uv7, &v3)
	FeMul(&uv7, &uv7, v)
	FeMul(&uv7, &uv7, u) // uv7 = uv^7

	FeSquare(&t0, &uv7)
	FeSquare(&t1, &t0)
	FeSquare(&t1, &t1)
	FeMul(&t1, &uv7, &t1)
	FeMul(&t0, &t0, &t1)
	FeSquare(&t0, &t0)
	FeMul(&t0, &t1, &t0)
	FeSquare(&t1, &t0)
	for i = 0; i < 4; i++ {
		FeSquare(&t1, &t1)
	}
	FeMul(&t0, &t1, &t0)
	FeSquare(&t1, &t0)
	for i = 0; i < 9; i++ {
		FeSquare(&t1, &t1)
	}
	FeMul(&t1, &t1, &t0)
	FeSquare(&t2, &t1)
	for i = 0; i < 19; i++ {
		FeSquare(&t2, &t2)
	}
	FeMul(&t1, &t2, &t1)
	for i = 0; i < 10; i++ {
		FeSquare(&t1, &t1)
	}
	FeMul(&t0, &t1, &t0)
	FeSquare(&t1, &t0)
	for i = 0; i < 49; i++ {
		FeSquare(&t1, &t1)
	}
	FeMul(&t1, &t1, &t0)
	FeSquare(&t2, &t1)
	for i = 0; i < 99; i++ {
		FeSquare(&t2, &t2)
	}
	FeMul(&t1, &t2, &t1)
	for i = 0; i < 50; i++ {
		FeSquare(&t1, &t1)
	}
	FeMul(&t0, &t1, &t0)
	FeSquare(&t0, &t0)
	FeSquare(&t0, &t0)
	FeMul(&t0, &t0, &uv7)

	/* End fe_pow22523.c */
	/* t0 = (uv^7)^((q-5)/8) */
	FeMul(&t0, &t0, &v3)
	FeMul(r, &t0, u) /* u^(m+1)v^(-(m+1)) */
}

func geFromfeFrombytesVartime(r *ProjectiveGroupElement, s *[32]byte) {
	var u, v, w, x, y, z FieldElement
	var sign byte

	h0 := load4(s[:])
	h1 := load3(s[4:]) << 6
	h2 := load3(s[7:]) << 5
	h3 := load3(s[10:]) << 3
	h4 := load3(s[13:]) << 2
	h5 := load4(s[16:])
	h6 := load3(s[20:]) << 7
	h7 := load3(s[23:]) << 5
	h8 := load3(s[26:]) << 4
	h9 := (load3(s[29:])) << 2
	FeCombine(&u, h0, h1, h2, h3, h4, h5, h6, h7, h8, h9)

	FeSquare2(&v, &u) // 2 * u^2
	FeOne(&w)
	FeAdd(&w, &v, &w)        // w = 2 * u^2 + 1
	FeSquare(&x, &w)         // w^2
	FeMul(&y, &feMA2, &v)    // -2 * A^2 * u^2
	FeAdd(&x, &x, &y)        // x = w^2 - 2 * A^2 * u^2
	FeDivPowm1(&r.X, &w, &x) // (w / x)^(m + 1)
	FeSquare(&y, &r.X)
	FeMul(&x, &y, &x)
	FeSub(&y, &w, &x)
	FeCopy(&z, &feMA)

	if checkFieldElement(&x, &y, &w, r) == 1 {
		FeMul(&x, &x, &SqrtM1)
		FeSub(&y, &w, &x)
		if FeIsNonZeroV1(&y) != 0 {
			FeMul(&r.X, &r.X, &feFFFB3)
		} else {
			FeMul(&r.X, &r.X, &feFFFB4)
		}
		// r->X = sqrt(A * (A + 2) * w / x)
		// z = -A
		sign = 1
	} else {
		FeMul(&r.X, &r.X, &u) // u * sqrt(2 * A * (A + 2) * w / x)
		FeMul(&z, &z, &v)     // -2 * A * u^2
		sign = 0
	}
	if FeIsNegativeV1(&r.X) != sign {
		FeNeg(&r.X, &r.X)
	}
	FeAdd(&r.Z, &z, &w)
	FeSub(&r.Y, &z, &w)
	FeMul(&r.X, &r.X, &r.Z)
}

//HashToEc ...
func HashToEc(key []byte, res *ExtendedGroupElement) {
	var point ProjectiveGroupElement
	var point2 CompletedGroupElement
	hash := sha3.KeccakSum256(key[:])
	geFromfeFrombytesVartime(&point, &hash)
	geMul8(&point2, &point)
	point2.ToExtended(res)
}

//CachedGroupElementCMove ...
func CachedGroupElementCMove(t, u *CachedGroupElement, b int32) {
	FeCMove(&t.yPlusX, &u.yPlusX, b)
	FeCMove(&t.yMinusX, &u.yMinusX, b)
	FeCMove(&t.Z, &u.Z, b)
	FeCMove(&t.T2d, &u.T2d, b)
}

func negative8(b int8) byte {
	x := uint64(b)
	x = x >> 63
	return byte(x)
}

//GeScalarMult Preconditions:
//   a[31] <= 127
func GeScalarMult(r *ProjectiveGroupElement, a *[32]byte, A *ExtendedGroupElement) {
	var e [64]int8
	var carry, carry2, i int
	var Ai [8]CachedGroupElement
	var t CompletedGroupElement
	var u ExtendedGroupElement

	carry = 0
	for i = 0; i < 31; i++ {
		carry += (int)(a[i])
		carry2 = (carry + 8) >> 4
		e[2*i] = (int8)(carry - (carry2 << 4))
		carry = (carry2 + 8) >> 4
		e[2*i+1] = (int8)(carry2 - (carry << 4))
	}
	carry += (int)(a[31])
	carry2 = (carry + 8) >> 4
	e[62] = (int8)(carry - (carry2 << 4))
	e[63] = (int8)(carry2)

	A.ToCached(&Ai[0])
	for i = 0; i < 7; i++ {
		GeAdd(&t, A, &Ai[i])
		t.ToExtended(&u)
		u.ToCached(&Ai[i+1])
	}

	r.Zero()
	for i = 63; i >= 0; i-- {
		b := e[i]
		bnegative := int8(negative8(b))
		babs := b - (((-bnegative) & b) << 1)
		var cur, minuscur CachedGroupElement
		r.Double(&t)
		t.ToProjective(r)
		r.Double(&t)
		t.ToProjective(r)
		r.Double(&t)
		t.ToProjective(r)
		r.Double(&t)
		t.ToExtended(&u)
		cur.Zero()
		for n := 0; n < 8; n++ {
			CachedGroupElementCMove(&cur, &Ai[n], equal(int32(babs), int32(n+1)))
		}
		FeCopy(&minuscur.yPlusX, &cur.yMinusX)
		FeCopy(&minuscur.yMinusX, &cur.yPlusX)
		FeCopy(&minuscur.Z, &cur.Z)
		FeNeg(&minuscur.T2d, &cur.T2d)
		CachedGroupElementCMove(&cur, &minuscur, int32(bnegative))
		GeAdd(&t, &u, &cur)
		t.ToProjective(r)
	}
}

//ScIsNonZero ...
func ScIsNonZero(s *[32]byte) int32 {
	var x uint8
	for _, b := range s {
		x |= b
	}
	x |= x >> 4
	x |= x >> 2
	x |= x >> 1
	return int32(x & 1)
}

//GeFromBytesVartime ...
func GeFromBytesVartime(p *ExtendedGroupElement, s *[32]byte) bool {
	var u, v, vxx, check FieldElement

	h0 := load4(s[:])
	h1 := load3(s[4:]) << 6
	h2 := load3(s[7:]) << 5
	h3 := load3(s[10:]) << 3
	h4 := load3(s[13:]) << 2
	h5 := load4(s[16:])
	h6 := load3(s[20:]) << 7
	h7 := load3(s[23:]) << 5
	h8 := load3(s[26:]) << 4
	h9 := (load3(s[29:]) & 8388607) << 2

	var carry0, carry1, carry2, carry3, carry4, carry5, carry6, carry7, carry8, carry9 int64
	// Validate the number to be canonical
	if h9 == 33554428 && h8 == 268435440 && h7 == 536870880 && h6 == 2147483520 &&
		h5 == 4294967295 && h4 == 67108860 && h3 == 134217720 && h2 == 536870880 &&
		h1 == 1073741760 && h0 >= 4294967277 {
		return false
	}
	carry9 = (h9 + (int64)(1<<24)) >> 25
	h0 += carry9 * 19
	h9 -= carry9 << 25
	carry1 = (h1 + (int64)(1<<24)) >> 25
	h2 += carry1
	h1 -= carry1 << 25
	carry3 = (h3 + (int64)(1<<24)) >> 25
	h4 += carry3
	h3 -= carry3 << 25
	carry5 = (h5 + (int64)(1<<24)) >> 25
	h6 += carry5
	h5 -= carry5 << 25
	carry7 = (h7 + (int64)(1<<24)) >> 25
	h8 += carry7
	h7 -= carry7 << 25
	carry0 = (h0 + (int64)(1<<25)) >> 26
	h1 += carry0
	h0 -= carry0 << 26
	carry2 = (h2 + (int64)(1<<25)) >> 26
	h3 += carry2
	h2 -= carry2 << 26
	carry4 = (h4 + (int64)(1<<25)) >> 26
	h5 += carry4
	h4 -= carry4 << 26
	carry6 = (h6 + (int64)(1<<25)) >> 26
	h7 += carry6
	h6 -= carry6 << 26
	carry8 = (h8 + (int64)(1<<25)) >> 26
	h9 += carry8
	h8 -= carry8 << 26

	p.Y[0] = int32(h0)
	p.Y[1] = int32(h1)
	p.Y[2] = int32(h2)
	p.Y[3] = int32(h3)
	p.Y[4] = int32(h4)
	p.Y[5] = int32(h5)
	p.Y[6] = int32(h6)
	p.Y[7] = int32(h7)
	p.Y[8] = int32(h8)
	p.Y[9] = int32(h9)

	FeOne(&p.Z)
	FeSquare(&u, &p.Y)
	FeMul(&v, &u, &d)
	FeSub(&u, &u, &p.Z) // y = y^2-1
	FeAdd(&v, &v, &p.Z) // v = dy^2+1

	FeDivPowm1(&p.X, &u, &v) // x = uv^3(uv^7)^((q-5)/8)

	FeSquare(&vxx, &p.X)
	FeMul(&vxx, &vxx, &v)   // v3 = v^3
	FeSub(&check, &vxx, &u) // vx^2-u
	if FeIsNonZeroV1(&check) != 0 {
		FeAdd(&check, &vxx, &u) // vx^2+u
		if FeIsNonZeroV1(&check) != 0 {
			return false
		}
		FeMul(&p.X, &p.X, &SqrtM1)
	}
	if FeIsNegativeV1(&p.X) != (s[31] >> 7) {
		if FeIsNonZeroV1(&p.X) == 0 {
			return false
		}
		FeNeg(&p.X, &p.X)
	}
	FeMul(&p.T, &p.X, &p.Y)
	return true
}

//GeDsmPrecomp ...
func GeDsmPrecomp(r *DsmPreCompGroupElement, s *ExtendedGroupElement) {
	var t CompletedGroupElement
	var s2, u ExtendedGroupElement
	s.ToCached(&r[0])
	s.Double(&t)
	t.ToExtended(&s2)

	for i := 0; i < 7; i++ {
		GeAdd(&t, &s2, &r[i])
		t.ToExtended(&u)
		u.ToCached(&r[i+1])
	}
}
