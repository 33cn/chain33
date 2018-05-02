package secp256k1_go

import (
	"encoding/hex"
	"fmt"
	"math/big"
)

type Field struct {
	n [10]uint32
}

func (a *Field) String() string {
	var tmp [32]byte
	b := *a
	b.Normalize()
	b.GetB32(tmp[:])
	return hex.EncodeToString(tmp[:])
}

func (a *Field) Print(lab string) {
	fmt.Println(lab+":", a.String())
}

func (a *Field) GetBig() (r *big.Int) {
	a.Normalize()
	r = new(big.Int)
	var tmp [32]byte
	a.GetB32(tmp[:])
	r.SetBytes(tmp[:])
	return
}

func (r *Field) SetB32(a []byte) {
	r.n[0] = 0
	r.n[1] = 0
	r.n[2] = 0
	r.n[3] = 0
	r.n[4] = 0
	r.n[5] = 0
	r.n[6] = 0
	r.n[7] = 0
	r.n[8] = 0
	r.n[9] = 0
	for i := uint(0); i < 32; i++ {
		for j := uint(0); j < 4; j++ {
			limb := (8*i + 2*j) / 26
			shift := (8*i + 2*j) % 26
			r.n[limb] |= (uint32)((a[31-i]>>(2*j))&0x3) << shift
		}
	}
}

func (r *Field) SetBytes(a []byte) {
	if len(a) > 32 {
		panic("too many bytes to set")
	}
	if len(a) == 32 {
		r.SetB32(a)
	} else {
		var buf [32]byte
		copy(buf[32-len(a):], a)
		r.SetB32(buf[:])
	}
}

func (r *Field) SetHex(s string) {
	d, _ := hex.DecodeString(s)
	r.SetBytes(d)
}

func (a *Field) IsOdd() bool {
	return (a.n[0] & 1) != 0
}

func (a *Field) IsZero() bool {
	return (a.n[0] == 0 && a.n[1] == 0 && a.n[2] == 0 && a.n[3] == 0 && a.n[4] == 0 && a.n[5] == 0 && a.n[6] == 0 && a.n[7] == 0 && a.n[8] == 0 && a.n[9] == 0)
}

func (r *Field) SetInt(a uint32) {
	r.n[0] = a
	r.n[1] = 0
	r.n[2] = 0
	r.n[3] = 0
	r.n[4] = 0
	r.n[5] = 0
	r.n[6] = 0
	r.n[7] = 0
	r.n[8] = 0
	r.n[9] = 0
}

func (r *Field) Normalize() {
	c := r.n[0]
	t0 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[1]
	t1 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[2]
	t2 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[3]
	t3 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[4]
	t4 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[5]
	t5 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[6]
	t6 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[7]
	t7 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[8]
	t8 := c & 0x3FFFFFF
	c = (c >> 26) + r.n[9]
	t9 := c & 0x03FFFFF
	c >>= 22

	// The following code will not modify the t's if c is initially 0.
	d := c*0x3D1 + t0
	t0 = d & 0x3FFFFFF
	d = (d >> 26) + t1 + c*0x40
	t1 = d & 0x3FFFFFF
	d = (d >> 26) + t2
	t2 = d & 0x3FFFFFF
	d = (d >> 26) + t3
	t3 = d & 0x3FFFFFF
	d = (d >> 26) + t4
	t4 = d & 0x3FFFFFF
	d = (d >> 26) + t5
	t5 = d & 0x3FFFFFF
	d = (d >> 26) + t6
	t6 = d & 0x3FFFFFF
	d = (d >> 26) + t7
	t7 = d & 0x3FFFFFF
	d = (d >> 26) + t8
	t8 = d & 0x3FFFFFF
	d = (d >> 26) + t9
	t9 = d & 0x03FFFFF

	// Subtract p if result >= p
	low := (uint64(t1) << 26) | uint64(t0)
	//mask := uint64(-(int64)((t9 < 0x03FFFFF) | (t8 < 0x3FFFFFF) | (t7 < 0x3FFFFFF) | (t6 < 0x3FFFFFF) | (t5 < 0x3FFFFFF) | (t4 < 0x3FFFFFF) | (t3 < 0x3FFFFFF) | (t2 < 0x3FFFFFF) | (low < 0xFFFFEFFFFFC2F)))
	var mask uint64
	if (t9 < 0x03FFFFF) ||
		(t8 < 0x3FFFFFF) ||
		(t7 < 0x3FFFFFF) ||
		(t6 < 0x3FFFFFF) ||
		(t5 < 0x3FFFFFF) ||
		(t4 < 0x3FFFFFF) ||
		(t3 < 0x3FFFFFF) ||
		(t2 < 0x3FFFFFF) ||
		(low < 0xFFFFEFFFFFC2F) {
		mask = 0xFFFFFFFFFFFFFFFF
	}
	t9 &= uint32(mask)
	t8 &= uint32(mask)
	t7 &= uint32(mask)
	t6 &= uint32(mask)
	t5 &= uint32(mask)
	t4 &= uint32(mask)
	t3 &= uint32(mask)
	t2 &= uint32(mask)
	low -= ((mask ^ 0xFFFFFFFFFFFFFFFF) & 0xFFFFEFFFFFC2F)

	// push internal variables back
	r.n[0] = uint32(low) & 0x3FFFFFF
	r.n[1] = uint32(low>>26) & 0x3FFFFFF
	r.n[2] = t2
	r.n[3] = t3
	r.n[4] = t4
	r.n[5] = t5
	r.n[6] = t6
	r.n[7] = t7
	r.n[8] = t8
	r.n[9] = t9
}

func (a *Field) GetB32(r []byte) {
	var i, j, c, limb, shift uint32
	for i = 0; i < 32; i++ {
		c = 0
		for j = 0; j < 4; j++ {
			limb = (8*i + 2*j) / 26
			shift = (8*i + 2*j) % 26
			c |= ((a.n[limb] >> shift) & 0x3) << (2 * j)
		}
		r[31-i] = byte(c)
	}
}

func (a *Field) Equals(b *Field) bool {
	return (a.n[0] == b.n[0] && a.n[1] == b.n[1] && a.n[2] == b.n[2] && a.n[3] == b.n[3] && a.n[4] == b.n[4] &&
		a.n[5] == b.n[5] && a.n[6] == b.n[6] && a.n[7] == b.n[7] && a.n[8] == b.n[8] && a.n[9] == b.n[9])
}

func (r *Field) SetAdd(a *Field) {
	r.n[0] += a.n[0]
	r.n[1] += a.n[1]
	r.n[2] += a.n[2]
	r.n[3] += a.n[3]
	r.n[4] += a.n[4]
	r.n[5] += a.n[5]
	r.n[6] += a.n[6]
	r.n[7] += a.n[7]
	r.n[8] += a.n[8]
	r.n[9] += a.n[9]
}

func (r *Field) MulInt(a uint32) {
	r.n[0] *= a
	r.n[1] *= a
	r.n[2] *= a
	r.n[3] *= a
	r.n[4] *= a
	r.n[5] *= a
	r.n[6] *= a
	r.n[7] *= a
	r.n[8] *= a
	r.n[9] *= a
}

func (a *Field) Negate(r *Field, m uint32) {
	r.n[0] = 0x3FFFC2F*(m+1) - a.n[0]
	r.n[1] = 0x3FFFFBF*(m+1) - a.n[1]
	r.n[2] = 0x3FFFFFF*(m+1) - a.n[2]
	r.n[3] = 0x3FFFFFF*(m+1) - a.n[3]
	r.n[4] = 0x3FFFFFF*(m+1) - a.n[4]
	r.n[5] = 0x3FFFFFF*(m+1) - a.n[5]
	r.n[6] = 0x3FFFFFF*(m+1) - a.n[6]
	r.n[7] = 0x3FFFFFF*(m+1) - a.n[7]
	r.n[8] = 0x3FFFFFF*(m+1) - a.n[8]
	r.n[9] = 0x03FFFFF*(m+1) - a.n[9]
}

/* New algo by peterdettman - https://github.com/sipa/TheCurve/pull/19 */
func (a *Field) Inv(r *Field) {
	var x2, x3, x6, x9, x11, x22, x44, x88, x176, x220, x223, t1 Field
	var j int

	a.Sqr(&x2)
	x2.Mul(&x2, a)

	x2.Sqr(&x3)
	x3.Mul(&x3, a)

	x3.Sqr(&x6)
	x6.Sqr(&x6)
	x6.Sqr(&x6)
	x6.Mul(&x6, &x3)

	x6.Sqr(&x9)
	x9.Sqr(&x9)
	x9.Sqr(&x9)
	x9.Mul(&x9, &x3)

	x9.Sqr(&x11)
	x11.Sqr(&x11)
	x11.Mul(&x11, &x2)

	x11.Sqr(&x22)
	for j = 1; j < 11; j++ {
		x22.Sqr(&x22)
	}
	x22.Mul(&x22, &x11)

	x22.Sqr(&x44)
	for j = 1; j < 22; j++ {
		x44.Sqr(&x44)
	}
	x44.Mul(&x44, &x22)

	x44.Sqr(&x88)
	for j = 1; j < 44; j++ {
		x88.Sqr(&x88)
	}
	x88.Mul(&x88, &x44)

	x88.Sqr(&x176)
	for j = 1; j < 88; j++ {
		x176.Sqr(&x176)
	}
	x176.Mul(&x176, &x88)

	x176.Sqr(&x220)
	for j = 1; j < 44; j++ {
		x220.Sqr(&x220)
	}
	x220.Mul(&x220, &x44)

	x220.Sqr(&x223)
	x223.Sqr(&x223)
	x223.Sqr(&x223)
	x223.Mul(&x223, &x3)

	x223.Sqr(&t1)
	for j = 1; j < 23; j++ {
		t1.Sqr(&t1)
	}
	t1.Mul(&t1, &x22)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Mul(&t1, a)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Mul(&t1, &x2)
	t1.Sqr(&t1)
	t1.Sqr(&t1)
	t1.Mul(r, a)
}

/* New algo by peterdettman - https://github.com/sipa/TheCurve/pull/19 */
func (a *Field) Sqrt(r *Field) {
	var x2, x3, x6, x9, x11, x22, x44, x88, x176, x220, x223, t1 Field
	var j int

	a.Sqr(&x2)
	x2.Mul(&x2, a)

	x2.Sqr(&x3)
	x3.Mul(&x3, a)

	x3.Sqr(&x6)
	x6.Sqr(&x6)
	x6.Sqr(&x6)
	x6.Mul(&x6, &x3)

	x6.Sqr(&x9)
	x9.Sqr(&x9)
	x9.Sqr(&x9)
	x9.Mul(&x9, &x3)

	x9.Sqr(&x11)
	x11.Sqr(&x11)
	x11.Mul(&x11, &x2)

	x11.Sqr(&x22)
	for j = 1; j < 11; j++ {
		x22.Sqr(&x22)
	}
	x22.Mul(&x22, &x11)

	x22.Sqr(&x44)
	for j = 1; j < 22; j++ {
		x44.Sqr(&x44)
	}
	x44.Mul(&x44, &x22)

	x44.Sqr(&x88)
	for j = 1; j < 44; j++ {
		x88.Sqr(&x88)
	}
	x88.Mul(&x88, &x44)

	x88.Sqr(&x176)
	for j = 1; j < 88; j++ {
		x176.Sqr(&x176)
	}
	x176.Mul(&x176, &x88)

	x176.Sqr(&x220)
	for j = 1; j < 44; j++ {
		x220.Sqr(&x220)
	}
	x220.Mul(&x220, &x44)

	x220.Sqr(&x223)
	x223.Sqr(&x223)
	x223.Sqr(&x223)
	x223.Mul(&x223, &x3)

	x223.Sqr(&t1)
	for j = 1; j < 23; j++ {
		t1.Sqr(&t1)
	}
	t1.Mul(&t1, &x22)
	for j = 0; j < 6; j++ {
		t1.Sqr(&t1)
	}
	t1.Mul(&t1, &x2)
	t1.Sqr(&t1)
	t1.Sqr(r)
}

func (a *Field) InvVar(r *Field) {
	var b [32]byte
	var c Field
	c = *a
	c.Normalize()
	c.GetB32(b[:])
	var n Number
	n.SetBytes(b[:])
	n.mod_inv(&n, &TheCurve.p)
	r.SetBytes(n.Bytes())
}

func (a *Field) Mul(r, b *Field) {
	var c, d uint64
	var t0, t1, t2, t3, t4, t5, t6 uint64
	var t7, t8, t9, t10, t11, t12, t13 uint64
	var t14, t15, t16, t17, t18, t19 uint64

	c = uint64(a.n[0]) * uint64(b.n[0])
	t0 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[1]) +
		uint64(a.n[1])*uint64(b.n[0])
	t1 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[2]) +
		uint64(a.n[1])*uint64(b.n[1]) +
		uint64(a.n[2])*uint64(b.n[0])
	t2 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[3]) +
		uint64(a.n[1])*uint64(b.n[2]) +
		uint64(a.n[2])*uint64(b.n[1]) +
		uint64(a.n[3])*uint64(b.n[0])
	t3 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[4]) +
		uint64(a.n[1])*uint64(b.n[3]) +
		uint64(a.n[2])*uint64(b.n[2]) +
		uint64(a.n[3])*uint64(b.n[1]) +
		uint64(a.n[4])*uint64(b.n[0])
	t4 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[5]) +
		uint64(a.n[1])*uint64(b.n[4]) +
		uint64(a.n[2])*uint64(b.n[3]) +
		uint64(a.n[3])*uint64(b.n[2]) +
		uint64(a.n[4])*uint64(b.n[1]) +
		uint64(a.n[5])*uint64(b.n[0])
	t5 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[6]) +
		uint64(a.n[1])*uint64(b.n[5]) +
		uint64(a.n[2])*uint64(b.n[4]) +
		uint64(a.n[3])*uint64(b.n[3]) +
		uint64(a.n[4])*uint64(b.n[2]) +
		uint64(a.n[5])*uint64(b.n[1]) +
		uint64(a.n[6])*uint64(b.n[0])
	t6 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[7]) +
		uint64(a.n[1])*uint64(b.n[6]) +
		uint64(a.n[2])*uint64(b.n[5]) +
		uint64(a.n[3])*uint64(b.n[4]) +
		uint64(a.n[4])*uint64(b.n[3]) +
		uint64(a.n[5])*uint64(b.n[2]) +
		uint64(a.n[6])*uint64(b.n[1]) +
		uint64(a.n[7])*uint64(b.n[0])
	t7 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[8]) +
		uint64(a.n[1])*uint64(b.n[7]) +
		uint64(a.n[2])*uint64(b.n[6]) +
		uint64(a.n[3])*uint64(b.n[5]) +
		uint64(a.n[4])*uint64(b.n[4]) +
		uint64(a.n[5])*uint64(b.n[3]) +
		uint64(a.n[6])*uint64(b.n[2]) +
		uint64(a.n[7])*uint64(b.n[1]) +
		uint64(a.n[8])*uint64(b.n[0])
	t8 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[0])*uint64(b.n[9]) +
		uint64(a.n[1])*uint64(b.n[8]) +
		uint64(a.n[2])*uint64(b.n[7]) +
		uint64(a.n[3])*uint64(b.n[6]) +
		uint64(a.n[4])*uint64(b.n[5]) +
		uint64(a.n[5])*uint64(b.n[4]) +
		uint64(a.n[6])*uint64(b.n[3]) +
		uint64(a.n[7])*uint64(b.n[2]) +
		uint64(a.n[8])*uint64(b.n[1]) +
		uint64(a.n[9])*uint64(b.n[0])
	t9 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[1])*uint64(b.n[9]) +
		uint64(a.n[2])*uint64(b.n[8]) +
		uint64(a.n[3])*uint64(b.n[7]) +
		uint64(a.n[4])*uint64(b.n[6]) +
		uint64(a.n[5])*uint64(b.n[5]) +
		uint64(a.n[6])*uint64(b.n[4]) +
		uint64(a.n[7])*uint64(b.n[3]) +
		uint64(a.n[8])*uint64(b.n[2]) +
		uint64(a.n[9])*uint64(b.n[1])
	t10 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[2])*uint64(b.n[9]) +
		uint64(a.n[3])*uint64(b.n[8]) +
		uint64(a.n[4])*uint64(b.n[7]) +
		uint64(a.n[5])*uint64(b.n[6]) +
		uint64(a.n[6])*uint64(b.n[5]) +
		uint64(a.n[7])*uint64(b.n[4]) +
		uint64(a.n[8])*uint64(b.n[3]) +
		uint64(a.n[9])*uint64(b.n[2])
	t11 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[3])*uint64(b.n[9]) +
		uint64(a.n[4])*uint64(b.n[8]) +
		uint64(a.n[5])*uint64(b.n[7]) +
		uint64(a.n[6])*uint64(b.n[6]) +
		uint64(a.n[7])*uint64(b.n[5]) +
		uint64(a.n[8])*uint64(b.n[4]) +
		uint64(a.n[9])*uint64(b.n[3])
	t12 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[4])*uint64(b.n[9]) +
		uint64(a.n[5])*uint64(b.n[8]) +
		uint64(a.n[6])*uint64(b.n[7]) +
		uint64(a.n[7])*uint64(b.n[6]) +
		uint64(a.n[8])*uint64(b.n[5]) +
		uint64(a.n[9])*uint64(b.n[4])
	t13 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[5])*uint64(b.n[9]) +
		uint64(a.n[6])*uint64(b.n[8]) +
		uint64(a.n[7])*uint64(b.n[7]) +
		uint64(a.n[8])*uint64(b.n[6]) +
		uint64(a.n[9])*uint64(b.n[5])
	t14 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[6])*uint64(b.n[9]) +
		uint64(a.n[7])*uint64(b.n[8]) +
		uint64(a.n[8])*uint64(b.n[7]) +
		uint64(a.n[9])*uint64(b.n[6])
	t15 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[7])*uint64(b.n[9]) +
		uint64(a.n[8])*uint64(b.n[8]) +
		uint64(a.n[9])*uint64(b.n[7])
	t16 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[8])*uint64(b.n[9]) +
		uint64(a.n[9])*uint64(b.n[8])
	t17 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[9])*uint64(b.n[9])
	t18 = c & 0x3FFFFFF
	c = c >> 26
	t19 = c

	c = t0 + t10*0x3D10
	t0 = c & 0x3FFFFFF
	c = c >> 26
	c = c + t1 + t10*0x400 + t11*0x3D10
	t1 = c & 0x3FFFFFF
	c = c >> 26
	c = c + t2 + t11*0x400 + t12*0x3D10
	t2 = c & 0x3FFFFFF
	c = c >> 26
	c = c + t3 + t12*0x400 + t13*0x3D10
	r.n[3] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t4 + t13*0x400 + t14*0x3D10
	r.n[4] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t5 + t14*0x400 + t15*0x3D10
	r.n[5] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t6 + t15*0x400 + t16*0x3D10
	r.n[6] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t7 + t16*0x400 + t17*0x3D10
	r.n[7] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t8 + t17*0x400 + t18*0x3D10
	r.n[8] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t9 + t18*0x400 + t19*0x1000003D10
	r.n[9] = uint32(c) & 0x03FFFFF
	c = c >> 22
	d = t0 + c*0x3D1
	r.n[0] = uint32(d) & 0x3FFFFFF
	d = d >> 26
	d = d + t1 + c*0x40
	r.n[1] = uint32(d) & 0x3FFFFFF
	d = d >> 26
	r.n[2] = uint32(t2 + d)
}

func (a *Field) Sqr(r *Field) {
	var c, d uint64
	var t0, t1, t2, t3, t4, t5, t6 uint64
	var t7, t8, t9, t10, t11, t12, t13 uint64
	var t14, t15, t16, t17, t18, t19 uint64

	c = uint64(a.n[0]) * uint64(a.n[0])
	t0 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[1])
	t1 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[2]) +
		uint64(a.n[1])*uint64(a.n[1])
	t2 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[3]) +
		(uint64(a.n[1])*2)*uint64(a.n[2])
	t3 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[4]) +
		(uint64(a.n[1])*2)*uint64(a.n[3]) +
		uint64(a.n[2])*uint64(a.n[2])
	t4 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[5]) +
		(uint64(a.n[1])*2)*uint64(a.n[4]) +
		(uint64(a.n[2])*2)*uint64(a.n[3])
	t5 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[6]) +
		(uint64(a.n[1])*2)*uint64(a.n[5]) +
		(uint64(a.n[2])*2)*uint64(a.n[4]) +
		uint64(a.n[3])*uint64(a.n[3])
	t6 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[7]) +
		(uint64(a.n[1])*2)*uint64(a.n[6]) +
		(uint64(a.n[2])*2)*uint64(a.n[5]) +
		(uint64(a.n[3])*2)*uint64(a.n[4])
	t7 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[8]) +
		(uint64(a.n[1])*2)*uint64(a.n[7]) +
		(uint64(a.n[2])*2)*uint64(a.n[6]) +
		(uint64(a.n[3])*2)*uint64(a.n[5]) +
		uint64(a.n[4])*uint64(a.n[4])
	t8 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[0])*2)*uint64(a.n[9]) +
		(uint64(a.n[1])*2)*uint64(a.n[8]) +
		(uint64(a.n[2])*2)*uint64(a.n[7]) +
		(uint64(a.n[3])*2)*uint64(a.n[6]) +
		(uint64(a.n[4])*2)*uint64(a.n[5])
	t9 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[1])*2)*uint64(a.n[9]) +
		(uint64(a.n[2])*2)*uint64(a.n[8]) +
		(uint64(a.n[3])*2)*uint64(a.n[7]) +
		(uint64(a.n[4])*2)*uint64(a.n[6]) +
		uint64(a.n[5])*uint64(a.n[5])
	t10 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[2])*2)*uint64(a.n[9]) +
		(uint64(a.n[3])*2)*uint64(a.n[8]) +
		(uint64(a.n[4])*2)*uint64(a.n[7]) +
		(uint64(a.n[5])*2)*uint64(a.n[6])
	t11 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[3])*2)*uint64(a.n[9]) +
		(uint64(a.n[4])*2)*uint64(a.n[8]) +
		(uint64(a.n[5])*2)*uint64(a.n[7]) +
		uint64(a.n[6])*uint64(a.n[6])
	t12 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[4])*2)*uint64(a.n[9]) +
		(uint64(a.n[5])*2)*uint64(a.n[8]) +
		(uint64(a.n[6])*2)*uint64(a.n[7])
	t13 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[5])*2)*uint64(a.n[9]) +
		(uint64(a.n[6])*2)*uint64(a.n[8]) +
		uint64(a.n[7])*uint64(a.n[7])
	t14 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[6])*2)*uint64(a.n[9]) +
		(uint64(a.n[7])*2)*uint64(a.n[8])
	t15 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[7])*2)*uint64(a.n[9]) +
		uint64(a.n[8])*uint64(a.n[8])
	t16 = c & 0x3FFFFFF
	c = c >> 26
	c = c + (uint64(a.n[8])*2)*uint64(a.n[9])
	t17 = c & 0x3FFFFFF
	c = c >> 26
	c = c + uint64(a.n[9])*uint64(a.n[9])
	t18 = c & 0x3FFFFFF
	c = c >> 26
	t19 = c

	c = t0 + t10*0x3D10
	t0 = c & 0x3FFFFFF
	c = c >> 26
	c = c + t1 + t10*0x400 + t11*0x3D10
	t1 = c & 0x3FFFFFF
	c = c >> 26
	c = c + t2 + t11*0x400 + t12*0x3D10
	t2 = c & 0x3FFFFFF
	c = c >> 26
	c = c + t3 + t12*0x400 + t13*0x3D10
	r.n[3] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t4 + t13*0x400 + t14*0x3D10
	r.n[4] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t5 + t14*0x400 + t15*0x3D10
	r.n[5] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t6 + t15*0x400 + t16*0x3D10
	r.n[6] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t7 + t16*0x400 + t17*0x3D10
	r.n[7] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t8 + t17*0x400 + t18*0x3D10
	r.n[8] = uint32(c) & 0x3FFFFFF
	c = c >> 26
	c = c + t9 + t18*0x400 + t19*0x1000003D10
	r.n[9] = uint32(c) & 0x03FFFFF
	c = c >> 22
	d = t0 + c*0x3D1
	r.n[0] = uint32(d) & 0x3FFFFFF
	d = d >> 26
	d = d + t1 + c*0x40
	r.n[1] = uint32(d) & 0x3FFFFFF
	d = d >> 26
	r.n[2] = uint32(t2 + d)
}
