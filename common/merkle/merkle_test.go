package merkle

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

const HashSize = 32

type Hash [HashSize]byte

const MaxHashStringSize = HashSize * 2

var ErrHashStrSize = fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)

// CloneBytes returns a copy of the bytes which represent the hash as a byte
// slice.
//
// NOTE: It is generally cheaper to just slice the hash directly thereby reusing
// the same bytes rather than calling this method.
func (hash *Hash) CloneBytes() []byte {
	newHash := make([]byte, HashSize)
	copy(newHash, hash[:])

	return newHash
}

// String returns the Hash as the hexadecimal string of the byte-reversed hash.
func (hash Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

// NewHash returns a new Hash from a byte slice.  An error is returned if
// the number of bytes passed in is not HashSize.
func NewHash(newHash []byte) (*Hash, error) {
	var sh Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not HashSize.
func (hash *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HashSize {
		return fmt.Errorf("invalid hash length of %v, want %v", nhlen,
			HashSize)
	}
	copy(hash[:], newHash)

	return nil
}

// NewHashFromStr creates a Hash from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Hash.
func NewHashFromStr(hash string) (*Hash, error) {
	ret := new(Hash)
	err := Decode(ret, hash)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Decode decodes the byte-reversed hexadecimal string encoding of a Hash to a
// destination.
func Decode(dst *Hash, src string) error {
	// Return error if hash string is too long.
	if len(src) > MaxHashStringSize {
		return ErrHashStrSize
	}

	// Hex decoder expects the hash to be a multiple of two.  When not, pad
	// with a leading zero.
	var srcBytes []byte
	if len(src)%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+len(src))
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}

	// Hex decode the source bytes to a temporary destination.
	var reversedHash Hash
	_, err := hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return err
	}

	// Reverse copy from the temporary hash to destination.  Because the
	// temporary was zeroed, the written result will be correctly padded.
	for i, b := range reversedHash[:HashSize/2] {
		dst[i], dst[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}

	return nil
}

//测试两个交易的roothash以及branch.获取bitcoin的99997 block作为验证
// "height": 99997,
//  "merkleroot": "5140e5972f672bf8e81bc189894c55a410723b095716eaeec845490aed785f0e",
//  "tx": [
//    "b86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c",
//    "80c6f121c3e9fe0a59177e49874d8c703cbadee0700a782e4002e87d862373c6"

func Test_TwoTxMerkle(t *testing.T) {
	RootHash := "5140e5972f672bf8e81bc189894c55a410723b095716eaeec845490aed785f0e"

	tx0string := "b86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c"
	tx1string := "80c6f121c3e9fe0a59177e49874d8c703cbadee0700a782e4002e87d862373c6"

	t.Logf("Test_TwoTxMerkle bitcoin roothash :%s", RootHash)

	rootHash, err := NewHashFromStr(RootHash)
	if err != nil {
		t.Errorf("NewHashFromStr RootHash err:%s", err.Error())
	}
	rootHashbyte := rootHash.CloneBytes()

	tx0hash, err := NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}
	tx0byte := tx0hash.CloneBytes()

	tx1hash, err := NewHashFromStr(tx1string)
	if err != nil {
		t.Errorf("NewHashFromStr tx1string err:%s", err.Error())
	}
	tx1byte := tx1hash.CloneBytes()

	leaves := make([][]byte, 2)

	leaves[0] = tx0byte
	leaves[1] = tx1byte

	bitroothash := GetMerkleRoot(leaves)

	bitroothashstr, err := NewHash(bitroothash)
	if err == nil {
		t.Logf("Test_TwoTxMerkle GetMerkleRoot roothash :%s", bitroothashstr.String())
	}
	if !bytes.Equal(rootHashbyte, bitroothash) {
		t.Errorf("Test_TwoTxMerkle  rootHashbyte :%v,GetMerkleRoot :%v", rootHashbyte, bitroothash)
		return
	}

	for txindex := 0; txindex < 2; txindex++ {
		roothashd, branchs := GetMerkleRootAndBranch(leaves, uint32(txindex))

		brroothashstr, err := NewHash(roothashd)
		if err == nil {
			t.Logf("Test_TwoTxMerkle GetMerkleRootAndBranch roothash :%s", brroothashstr.String())
		}

		brroothash := GetMerkleRootFromBranch(branchs, leaves[txindex], uint32(txindex))
		if bytes.Equal(bitroothash, brroothash) && bytes.Equal(rootHashbyte, brroothash) {
			brRoothashstr, err := NewHash(brroothash)
			if err == nil {
				t.Logf("Test_TwoTxMerkle GetMerkleRootFromBranch roothash :%s", brRoothashstr.String())
			}
			t.Logf("Test_TwoTxMerkle bitroothash == brroothash :%d", txindex)
		}
	}
}

//测试三个交易的roothash以及branch
//"height": 99960,
//  "merkleroot": "34d5a57822efa653019edfee29b9586a0d0d807572275b45f39a7e9c25614bf9",
//  "tx": [
//    "f89c65bdcd695e4acc621256085f20d7c093097e04a1ce34b606a5829cbaf2c6",
//    "1818bef9c6aeed09de0ed999b5f2868b3555084437e1c63f29d5f37b69bb214f",
//    "d43a40a2db5bad2bd176c27911ed86d97bff734425953b19c8cf77910b21020d"
func Test_OddTxMerkle(t *testing.T) {

	RootHash := "34d5a57822efa653019edfee29b9586a0d0d807572275b45f39a7e9c25614bf9"

	tx0string := "f89c65bdcd695e4acc621256085f20d7c093097e04a1ce34b606a5829cbaf2c6"
	tx1string := "1818bef9c6aeed09de0ed999b5f2868b3555084437e1c63f29d5f37b69bb214f"
	tx2string := "d43a40a2db5bad2bd176c27911ed86d97bff734425953b19c8cf77910b21020d"

	t.Logf("Test_OddTxMerkle bitcoin roothash :%s", RootHash)

	rootHash, err := NewHashFromStr(RootHash)
	if err != nil {
		t.Errorf("NewHashFromStr RootHash err:%s", err.Error())
	}
	rootHashbyte := rootHash.CloneBytes()

	tx0hash, err := NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}
	tx0byte := tx0hash.CloneBytes()

	tx1hash, err := NewHashFromStr(tx1string)
	if err != nil {
		t.Errorf("NewHashFromStr tx1string err:%s", err.Error())
	}
	tx1byte := tx1hash.CloneBytes()

	tx2hash, err := NewHashFromStr(tx2string)
	if err != nil {
		t.Errorf("NewHashFromStr tx2string err:%s", err.Error())
	}
	tx2byte := tx2hash.CloneBytes()

	leaves := make([][]byte, 3)

	leaves[0] = tx0byte
	leaves[1] = tx1byte
	leaves[2] = tx2byte

	bitroothash := GetMerkleRoot(leaves)

	bitroothashstr, err := NewHash(bitroothash)
	if err == nil {
		t.Logf("Test_OddTxMerkle GetMerkleRoot roothash :%s", bitroothashstr.String())
	}
	if !bytes.Equal(rootHashbyte, bitroothash) {
		t.Errorf("Test_OddTxMerkle  rootHashbyte :%v,GetMerkleRoot :%v", rootHashbyte, bitroothash)
		return
	}

	for txindex := 0; txindex < 3; txindex++ {
		branchs := GetMerkleBranch(leaves, uint32(txindex))

		brroothash := GetMerkleRootFromBranch(branchs, leaves[txindex], uint32(txindex))
		if bytes.Equal(bitroothash, brroothash) && bytes.Equal(rootHashbyte, brroothash) {
			brRoothashstr, err := NewHash(brroothash)
			if err == nil {
				t.Logf("Test_OddTxMerkle GetMerkleRootFromBranch roothash :%s", brRoothashstr.String())
			}
			t.Logf("Test_OddTxMerkle bitroothash == brroothash :%d", txindex)
		}
	}
}

//测试六个交易的roothash以及branch
//		"height": 99974,
//  	"merkleroot": "272066470ccf8ee2bb6f48f263c5b2ffc56813be40001c4b33fec0677d69e3cc",
//  	"tx": [
//    "dc5d3f45eeffaab3a71e9930c80a34f5da7ee4cdbc6e7270e1c6d3397f835836",
//    "0ed98b4429bfbf78cfb1e87b9c16cc6641df0a8b2df861c13dc987785912cc48",
//    "8108da17be5960b009de33b60fcdfd5856abbd8b7c543afde31b0601b6e2aeea",
//    "ecffa48ff29b13b295c25d97987f4899ed4f607691c316b0fad685fa1ab268d9",
//    "b57f79efed1d0e999495a34840aa7b41e15b43627225849d83effe56152df4e7",
//    "8ded96ae2a609555df2390faa2f001b04e6e0ba8a60c69f9b432c9637157d766"

func Test_SixTxMerkle(t *testing.T) {
	RootHash := "272066470ccf8ee2bb6f48f263c5b2ffc56813be40001c4b33fec0677d69e3cc"

	tx0string := "dc5d3f45eeffaab3a71e9930c80a34f5da7ee4cdbc6e7270e1c6d3397f835836"
	tx1string := "0ed98b4429bfbf78cfb1e87b9c16cc6641df0a8b2df861c13dc987785912cc48"
	tx2string := "8108da17be5960b009de33b60fcdfd5856abbd8b7c543afde31b0601b6e2aeea"
	tx3string := "ecffa48ff29b13b295c25d97987f4899ed4f607691c316b0fad685fa1ab268d9"
	tx4string := "b57f79efed1d0e999495a34840aa7b41e15b43627225849d83effe56152df4e7"
	tx5string := "8ded96ae2a609555df2390faa2f001b04e6e0ba8a60c69f9b432c9637157d766"

	t.Logf("Test_SixTxMerkle bitcoin roothash :%s", RootHash)

	rootHash, err := NewHashFromStr(RootHash)
	if err != nil {
		t.Errorf("NewHashFromStr RootHash err:%s", err.Error())
	}
	rootHashbyte := rootHash.CloneBytes()

	tx0hash, err := NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}
	tx0byte := tx0hash.CloneBytes()

	tx1hash, err := NewHashFromStr(tx1string)
	if err != nil {
		t.Errorf("NewHashFromStr tx1string err:%s", err.Error())
	}
	tx1byte := tx1hash.CloneBytes()

	tx2hash, err := NewHashFromStr(tx2string)
	if err != nil {
		t.Errorf("NewHashFromStr tx2string err:%s", err.Error())
	}
	tx2byte := tx2hash.CloneBytes()

	tx3hash, err := NewHashFromStr(tx3string)
	if err != nil {
		t.Errorf("NewHashFromStr tx3string err:%s", err.Error())
	}
	tx3byte := tx3hash.CloneBytes()

	tx4hash, err := NewHashFromStr(tx4string)
	if err != nil {
		t.Errorf("NewHashFromStr tx4string err:%s", err.Error())
	}
	tx4byte := tx4hash.CloneBytes()

	tx5hash, err := NewHashFromStr(tx5string)
	if err != nil {
		t.Errorf("NewHashFromStr tx5string err:%s", err.Error())
	}
	tx5byte := tx5hash.CloneBytes()

	leaves := make([][]byte, 6)

	leaves[0] = tx0byte
	leaves[1] = tx1byte
	leaves[2] = tx2byte
	leaves[3] = tx3byte

	leaves[4] = tx4byte
	leaves[5] = tx5byte

	bitroothash := GetMerkleRoot(leaves)

	bitroothashstr, err := NewHash(bitroothash)
	if err == nil {
		t.Logf("Test_SixTxMerkle GetMerkleRoot roothash :%s", bitroothashstr.String())
	}
	if !bytes.Equal(rootHashbyte, bitroothash) {
		t.Errorf("Test_SixTxMerkle  rootHashbyte :%v,GetMerkleRoot :%v", rootHashbyte, bitroothash)
		return
	}

	for txindex := 0; txindex < 6; txindex++ {
		branchs := GetMerkleBranch(leaves, uint32(txindex))

		brroothash := GetMerkleRootFromBranch(branchs, leaves[txindex], uint32(txindex))
		if bytes.Equal(bitroothash, brroothash) && bytes.Equal(rootHashbyte, brroothash) {
			brRoothashstr, err := NewHash(brroothash)
			if err == nil {
				t.Logf("Test_SixTxMerkle GetMerkleRootFromBranch roothash :%s", brRoothashstr.String())
			}
			t.Logf("Test_SixTxMerkle bitroothash == brroothash :%d", txindex)
		}
	}
}
