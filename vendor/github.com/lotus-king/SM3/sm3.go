package sm3

import (
	"encoding/binary"
	"hash"
)

func init() {
	//To do...
	//crypto.RegisterHash(crypto.SM3, New)
}

type SM3 struct {
	digest      [8]uint32  // digest represents the partial evaluation of V
	T           [64]uint32 // constant
	length      uint64     // length of the message
	unhandleMsg []byte     // uint8  //
}

// The size of a SM3 checksum in bytes.
const Size = 32

// The blocksize of SM3 in bytes.
const BlockSize = 64

func New() hash.Hash {
	sm3 := new(SM3)
	sm3.Reset()
	return sm3
}

// Reset clears the internal state by zeroing bytes in the state buffer.
// This can be skipped for a newly-created hash state; the default zero-allocated state is correct.
func (sm3 *SM3) Reset() {
	// Reset digest
	sm3.digest[0] = 0x7380166f
	sm3.digest[1] = 0x4914b2b9
	sm3.digest[2] = 0x172442d7
	sm3.digest[3] = 0xda8a0600
	sm3.digest[4] = 0xa96f30bc
	sm3.digest[5] = 0x163138aa
	sm3.digest[6] = 0xe38dee4d
	sm3.digest[7] = 0xb0fb0e4e

	// Set T[i]
	for i := 0; i < 16; i++ {
		sm3.T[i] = 0x79cc4519
	}
	for i := 16; i < 64; i++ {
		sm3.T[i] = 0x7a879d8a
	}
	// Reset numberic states
	sm3.length = 0

	sm3.unhandleMsg = []byte{}
}

// BlockSize, required by the hash.Hash interface.
// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (sm3 *SM3) BlockSize() int {
	// Here return the number of byte
	return BlockSize
}

// Size, required by the hash.Hash interface.
// Size returns the number of bytes Sum will return.
func (sm3 *SM3) Size() int {
	return Size
}

func (sm3 *SM3) ff0(x, y, z uint32) uint32 {
	return x ^ y ^ z
}

func (sm3 *SM3) ff1(x, y, z uint32) uint32 {
	return (x & y) | (x & z) | (y & z)
}

func (sm3 *SM3) gg0(x, y, z uint32) uint32 {
	return x ^ y ^ z
}

func (sm3 *SM3) gg1(x, y, z uint32) uint32 {
	return (x & y) | (^x & z)
}

func (sm3 *SM3) p0(x uint32) uint32 {
	return x ^ sm3.leftRotate(x, 9) ^ sm3.leftRotate(x, 17)
}

func (sm3 *SM3) p1(x uint32) uint32 {
	return x ^ sm3.leftRotate(x, 15) ^ sm3.leftRotate(x, 23)
}

func (sm3 *SM3) messageExtend(data []byte) (W [68]uint32, W1 [64]uint32) {

	// big endian
	for i := 0; i < 16; i++ {
		W[i] = binary.BigEndian.Uint32(data[4*i : 4*(i+1)])
	}
	for i := 16; i < 68; i++ {
		W[i] = sm3.p1(W[i-16]^W[i-9]^sm3.leftRotate(W[i-3], 15)) ^ sm3.leftRotate(W[i-13], 7) ^ W[i-6]
	}
	for i := 0; i < 64; i++ {
		W1[i] = W[i] ^ W[i+4]
	}
	return W, W1
}

func (sm3 *SM3) leftRotate(x uint32, i uint32) uint32 {
	i %= 32
	return (x<<i | x>>(32-i))
}

// cf is compress function
func (sm3 *SM3) cf(W [68]uint32, W1 [64]uint32) {

	A := sm3.digest[0]
	B := sm3.digest[1]
	C := sm3.digest[2]
	D := sm3.digest[3]
	E := sm3.digest[4]
	F := sm3.digest[5]
	G := sm3.digest[6]
	H := sm3.digest[7]

	for i := 0; i < 16; i++ {
		SS1 := sm3.leftRotate(sm3.leftRotate(A, 12)+E+sm3.leftRotate(sm3.T[i], uint32(i)), 7)
		SS2 := SS1 ^ sm3.leftRotate(A, 12)
		TT1 := sm3.ff0(A, B, C) + D + SS2 + W1[i]
		TT2 := sm3.gg0(E, F, G) + H + SS1 + W[i]
		D = C
		C = sm3.leftRotate(B, 9)
		B = A
		A = TT1
		H = G
		G = sm3.leftRotate(F, 19)
		F = E
		E = sm3.p0(TT2)
	}

	for i := 16; i < 64; i++ {
		SS1 := sm3.leftRotate(sm3.leftRotate(A, 12)+E+sm3.leftRotate(sm3.T[i], uint32(i)), 7)
		SS2 := SS1 ^ sm3.leftRotate(A, 12)
		TT1 := sm3.ff1(A, B, C) + D + SS2 + W1[i]
		TT2 := sm3.gg1(E, F, G) + H + SS1 + W[i]
		D = C
		C = sm3.leftRotate(B, 9)
		B = A
		A = TT1
		H = G
		G = sm3.leftRotate(F, 19)
		F = E
		E = sm3.p0(TT2)
	}

	sm3.digest[0] ^= A
	sm3.digest[1] ^= B
	sm3.digest[2] ^= C
	sm3.digest[3] ^= D
	sm3.digest[4] ^= E
	sm3.digest[5] ^= F
	sm3.digest[6] ^= G
	sm3.digest[7] ^= H
}

// update, iterative compress, update digests
func (sm3 *SM3) update(msg []byte, nblocks int) {
	for i := 0; i < nblocks; i++ {
		startPos := i * sm3.BlockSize()
		W, W1 := sm3.messageExtend(msg[startPos : startPos+sm3.BlockSize()])

		sm3.cf(W, W1)
	}
}

// Write, required by the hash.Hash interface.
// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (sm3 *SM3) Write(p []byte) (int, error) {

	toWrite := len(p)
	sm3.length += uint64(len(p) * 8)

	msg := append(sm3.unhandleMsg, p...)
	nblocks := len(msg) / sm3.BlockSize()
	sm3.update(msg, nblocks)

	// Update unhandleMsg
	sm3.unhandleMsg = msg[nblocks*sm3.BlockSize():]

	return toWrite, nil
}

func (sm3 *SM3) pad() []byte {

	// Make a copy not using unhandleMsg
	msg := sm3.unhandleMsg

	// Append '1'
	msg = append(msg, 0x80)

	// Append until the resulting message length (in bits) is congruent to 448 (mod 512)
	blockSize := 64
	for len(msg)%blockSize != 56 {
		msg = append(msg, 0x00)
	}

	// append message length
	msg = append(msg, uint8(sm3.length>>56&0xff))
	msg = append(msg, uint8(sm3.length>>48&0xff))
	msg = append(msg, uint8(sm3.length>>40&0xff))
	msg = append(msg, uint8(sm3.length>>32&0xff))
	msg = append(msg, uint8(sm3.length>>24&0xff))
	msg = append(msg, uint8(sm3.length>>16&0xff))
	msg = append(msg, uint8(sm3.length>>8&0xff))
	msg = append(msg, uint8(sm3.length>>0&0xff))

	if len(msg)%64 != 0 {
		panic("SM3 pad: Error")
	}

	return msg
}

// Sum, required by the hash.Hash interface.
// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (sm3 *SM3) Sum(in []byte) []byte {
	cpsm3 := *sm3
	hash := cpsm3.checkSum()

	return append(in, hash[:]...)
}
func (sm3 *SM3) checkSum() [Size]byte {

	msg := sm3.pad()

	// Finialize
	sm3.update(msg, len(msg)/sm3.BlockSize())

	var out [Size]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(out[i*4:], sm3.digest[i])
	}
	return out
}

func Sum(in []byte) [Size]byte {
	var sm3 SM3
	sm3.Reset()
	sm3.Write(in[:])
	return sm3.checkSum()
}
