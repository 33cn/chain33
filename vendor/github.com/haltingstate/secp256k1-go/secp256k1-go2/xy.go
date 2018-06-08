package secp256k1_go

import (
	"fmt"
	"log"
)

type XY struct {
	X, Y     Field
	Infinity bool
}

func (ge *XY) Print(lab string) {
	if ge.Infinity {
		fmt.Println(lab + " - Infinity")
		return
	}
	fmt.Println(lab+".X:", ge.X.String())
	fmt.Println(lab+".Y:", ge.Y.String())
}

//edited

/*
   if (size == 33 && (pub[0] == 0x02 || pub[0] == 0x03)) {
       secp256k1_fe_t x;
       secp256k1_fe_set_b32(&x, pub+1);
       return secp256k1_ge_set_xo(elem, &x, pub[0] == 0x03);
   } else if (size == 65 && (pub[0] == 0x04 || pub[0] == 0x06 || pub[0] == 0x07)) {
       secp256k1_fe_t x, y;
       secp256k1_fe_set_b32(&x, pub+1);
       secp256k1_fe_set_b32(&y, pub+33);
       secp256k1_ge_set_xy(elem, &x, &y);
       if ((pub[0] == 0x06 || pub[0] == 0x07) && secp256k1_fe_is_odd(&y) != (pub[0] == 0x07))
           return 0;
       return secp256k1_ge_is_valid(elem);
   }
*/
//All compact keys appear to be valid by construction, but may fail
//is valid check

//WARNING: for compact signatures, will succeed unconditionally
//however, elem.IsValid will fail
func (elem *XY) ParsePubkey(pub []byte) bool {
	if len(pub) != 33 {
		log.Panic()
	}
	if len(pub) == 33 && (pub[0] == 0x02 || pub[0] == 0x03) {
		elem.X.SetB32(pub[1:33])
		elem.SetXO(&elem.X, pub[0] == 0x03)
	} else {
		log.Panic()
		return false
	}
	//THIS FAILS
	//reenable later
	//if elem.IsValid() == false {
	//	return false
	//}

	/*
		 else if len(pub) == 65 && (pub[0] == 0x04 || pub[0] == 0x06 || pub[0] == 0x07) {
			elem.X.SetB32(pub[1:33])
			elem.Y.SetB32(pub[33:65])
			if (pub[0] == 0x06 || pub[0] == 0x07) && elem.Y.IsOdd() != (pub[0] == 0x07) {
				return false
			}
		}
	*/
	return true
}

// Returns serialized key in in compressed format: "<02> <X>",
// eventually "<03> <X>"
//33 bytes
func (pub XY) Bytes() []byte {
	pub.X.Normalize() // See GitHub issue #15

	var raw []byte = make([]byte, 33)
	if pub.Y.IsOdd() {
		raw[0] = 0x03
	} else {
		raw[0] = 0x02
	}
	pub.X.GetB32(raw[1:])
	return raw
}

// Returns serialized key in uncompressed format "<04> <X> <Y>"
//65 bytes
func (pub *XY) BytesUncompressed() (raw []byte) {
	pub.X.Normalize() // See GitHub issue #15
	pub.Y.Normalize() // See GitHub issue #15

	raw = make([]byte, 65)
	raw[0] = 0x04
	pub.X.GetB32(raw[1:33])
	pub.Y.GetB32(raw[33:65])
	return
}

func (r *XY) SetXY(X, Y *Field) {
	r.Infinity = false
	r.X = *X
	r.Y = *Y
}

/*
int static secp256k1_ecdsa_pubkey_parse(secp256k1_ge_t *elem, const unsigned char *pub, int size) {
    if (size == 33 && (pub[0] == 0x02 || pub[0] == 0x03)) {
        secp256k1_fe_t x;
        secp256k1_fe_set_b32(&x, pub+1);
        return secp256k1_ge_set_xo(elem, &x, pub[0] == 0x03);
    } else if (size == 65 && (pub[0] == 0x04 || pub[0] == 0x06 || pub[0] == 0x07)) {
        secp256k1_fe_t x, y;
        secp256k1_fe_set_b32(&x, pub+1);
        secp256k1_fe_set_b32(&y, pub+33);
        secp256k1_ge_set_xy(elem, &x, &y);
        if ((pub[0] == 0x06 || pub[0] == 0x07) && secp256k1_fe_is_odd(&y) != (pub[0] == 0x07))
            return 0;
        return secp256k1_ge_is_valid(elem);
    } else {
        return 0;
    }
}
*/

//    if (size == 33 && (pub[0] == 0x02 || pub[0] == 0x03)) {
//        secp256k1_fe_t x;
//        secp256k1_fe_set_b32(&x, pub+1);
//        return secp256k1_ge_set_xo(elem, &x, pub[0] == 0x03);

func (a *XY) IsValid() bool {
	if a.Infinity {
		return false
	}
	var y2, x3, c Field
	a.Y.Sqr(&y2)
	a.X.Sqr(&x3)
	x3.Mul(&x3, &a.X)
	c.SetInt(7)
	x3.SetAdd(&c)
	y2.Normalize()
	x3.Normalize()
	return y2.Equals(&x3)
}

func (r *XY) SetXYZ(a *XYZ) {
	var z2, z3 Field
	a.Z.InvVar(&a.Z)
	a.Z.Sqr(&z2)
	a.Z.Mul(&z3, &z2)
	a.X.Mul(&a.X, &z2)
	a.Y.Mul(&a.Y, &z3)
	a.Z.SetInt(1)
	r.Infinity = a.Infinity
	r.X = a.X
	r.Y = a.Y
}

func (a *XY) precomp(w int) (pre []XY) {
	pre = make([]XY, (1 << (uint(w) - 2)))
	pre[0] = *a
	var X, d, tmp XYZ
	X.SetXY(a)
	X.Double(&d)
	for i := 1; i < len(pre); i++ {
		d.AddXY(&tmp, &pre[i-1])
		pre[i].SetXYZ(&tmp)
	}
	return
}

func (a *XY) Neg(r *XY) {
	r.Infinity = a.Infinity
	r.X = a.X
	r.Y = a.Y
	r.Y.Normalize()
	r.Y.Negate(&r.Y, 1)
}

/*
int static secp256k1_ge_set_xo(secp256k1_ge_t *r, const secp256k1_fe_t *x, int odd) {
    r->x = *x;
    secp256k1_fe_t x2; secp256k1_fe_sqr(&x2, x);
    secp256k1_fe_t x3; secp256k1_fe_mul(&x3, x, &x2);
    r->infinity = 0;
    secp256k1_fe_t c; secp256k1_fe_set_int(&c, 7);
    secp256k1_fe_add(&c, &x3);
    if (!secp256k1_fe_sqrt(&r->y, &c))
        return 0;
    secp256k1_fe_normalize(&r->y);
    if (secp256k1_fe_is_odd(&r->y) != odd)
        secp256k1_fe_negate(&r->y, &r->y, 1);
    return 1;
}
*/

func (r *XY) SetXO(X *Field, odd bool) {
	var c, x2, x3 Field
	r.X = *X
	X.Sqr(&x2)
	X.Mul(&x3, &x2)
	r.Infinity = false
	c.SetInt(7)
	c.SetAdd(&x3)
	c.Sqrt(&r.Y) //does not return, can fail
	if r.Y.IsOdd() != odd {
		r.Y.Negate(&r.Y, 1)
	}

	//r.X.Normalize() // See GitHub issue #15
	r.Y.Normalize()
}

func (pk *XY) AddXY(a *XY) {
	var xyz XYZ
	xyz.SetXY(pk)
	xyz.AddXY(&xyz, a)
	pk.SetXYZ(&xyz)
}

/*
func (pk *XY) GetPublicKey() []byte {
	var out []byte = make([]byte, 65, 65)
	pk.X.GetB32(out[1:33])
	if len(out) == 65 {
		out[0] = 0x04
		pk.Y.GetB32(out[33:65])
	} else {
		if pk.Y.IsOdd() {
			out[0] = 0x03
		} else {
			out[0] = 0x02
		}
	}
	return out
}
*/

//use compact format
//returns only 33 bytes
//same as bytes()
//TODO: deprecate, replace with .Bytes()
func (pk *XY) GetPublicKey() []byte {
	return pk.Bytes()
	/*
		var out []byte = make([]byte, 33, 33)
		pk.X.GetB32(out[1:33])
		if pk.Y.IsOdd() {
			out[0] = 0x03
		} else {
			out[0] = 0x02
		}
		return out
	*/
}
