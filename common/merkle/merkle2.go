package merkle

import "bytes"
import "crypto/sha256"

func CalcMerkle(mtr [][]byte) (res []byte, mutated bool) {
	var j, i2 int
	for siz := len(mtr); siz > 1; siz = (siz + 1) / 2 {
		for i := 0; i < siz; i += 2 {
			if i+1 < siz-1 {
				i2 = i + 1
			} else {
				i2 = siz - 1
			}
			if i != i2 && bytes.Equal(mtr[j+i], mtr[j+i2]) {
				mutated = true
			}
			s := sha256.New()
			s.Write(mtr[j+i])
			s.Write(mtr[j+i2])
			tmp := s.Sum(nil)
			s.Reset()
			s.Write(tmp)
			mtr = append(mtr, s.Sum(nil))
		}
		j += siz
	}
	res = mtr[len(mtr)-1]
	return
}
