package secp256k1

import (
	crand "crypto/rand" //secure, system random number generator
	"crypto/sha256"
	"hash"
	"io"
	"log"
	//mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	sha256Hash hash.Hash = sha256.New()
)

func SumSHA256(b []byte) []byte {
	sha256Hash.Reset()
	sha256Hash.Write(b)
	sum := sha256Hash.Sum(nil)
	return sum[:]
}

/*
Entropy pool needs
- state (an array of bytes)
- a compression function (two 256 bit blocks to single block)
- a mixing function across the pool

- Xor is safe, as it cannot make value less random
-- apply compression function, then xor with current value
--

*/

type EntropyPool struct {
	Ent [32]byte //256 bit accumulator

}

//mixes in 256 bits, outputs 256 bits
func (self *EntropyPool) Mix256(in []byte) (out []byte) {

	//hash input
	val1 := SumSHA256(in)
	//return value
	val2 := SumSHA256(append(val1, self.Ent[:]...))
	//next ent value
	val3 := SumSHA256(append(val1, val2...))

	for i := 0; i < 32; i++ {
		self.Ent[i] = val3[i]
		val3[i] = 0x00
	}

	return val2
}

//take in N bytes, salts, return N
func (self *EntropyPool) Mix(in []byte) []byte {
	var length int = len(in) - len(in)%32 + 32
	var buff []byte = make([]byte, length, length)
	for i := 0; i < len(in); i++ {
		buff[i] = in[i]
	}
	var iterations int = (len(in) / 32) + 1
	for i := 0; i < iterations; i++ {
		tmp := self.Mix256(buff[32*i : 32+32*i]) //32 byte slice
		for j := 0; j < 32; j++ {
			buff[i*32+j] = tmp[j]
		}
	}
	return buff[:len(in)]
}

/*
Note:

- On windows cryto/rand uses CrytoGenRandom which uses RC4 which is insecure
- Android random number generator is known to be insecure.
- Linux uses /dev/urandom , which is thought to be secure and uses entropy pool

Therefore the output is salted.
*/

/*
Note:

Should allow pseudo-random mode for repeatability for certain types of tests

*/

//var _rand *mrand.Rand //pseudorandom number generator
var _ent EntropyPool

//seed pseudo random number generator with
// hash of system time in nano seconds
// hash of system environmental variables
// hash of process id
func init() {
	var seed1 []byte = []byte(strconv.FormatUint(uint64(time.Now().UnixNano()), 16))
	var seed2 []byte = []byte(strings.Join(os.Environ(), ""))
	var seed3 []byte = []byte(strconv.FormatUint(uint64(os.Getpid()), 16))

	seed4 := make([]byte, 256)
	io.ReadFull(crand.Reader, seed4) //system secure random number generator

	//mrand.Rand_rand = mrand.New(mrand.NewSource(int64(time.Now().UnixNano()))) //pseudo random
	//seed entropy pool
	_ent.Mix256(seed1)
	_ent.Mix256(seed2)
	_ent.Mix256(seed3)
	_ent.Mix256(seed4)
}

//Secure Random number generator for forwards security
//On Unix-like systems, Reader reads from /dev/urandom.
//On Windows systems, Reader uses the CryptGenRandom API.
//Pseudo-random sequence, seeded from program start time, environmental variables,
//and process id is mixed in for forward security. Future version should use entropy pool
func RandByte(n int) []byte {
	buff := make([]byte, n)
	ret, err := io.ReadFull(crand.Reader, buff) //system secure random number generator
	if len(buff) != ret || err != nil {
		log.Panic()
	}

	//XORing in sequence, cannot reduce security (even if sequence is bad/known/non-random)

	buff2 := _ent.Mix(buff)
	for i := 0; i < n; i++ {
		buff[i] ^= buff2[i]
	}
	return buff
}
