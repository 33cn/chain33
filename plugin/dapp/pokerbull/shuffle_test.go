package pokerbull

import (
	"testing"
	"math/rand"
	"fmt"
	"strconv"
)

func shuffle(attr []int, src int64) []int {
	length := len(attr)
	if length == 0 {
		fmt.Println("Invalid Parameter.")
		return nil
	}

	var attrV = make([]int, length)
	copy(attrV, attr)

	rng := rand.New(rand.NewSource(src))
	for i := 0; i < length; i++ {
		idx := rng.Intn(length-1)
		tmpV := attrV[idx]
		attrV[idx] = attrV[length-i-1]
		attrV[length-i-1] = tmpV
	}

	return attrV
}

func TestShuffle(t *testing.T) {
	var attr = []int{0,1,2,3,4,5,6,7,8,9}
	var res = []int{0,0,0,0,0,0,0,0,0,0}

	for j := 0; j < 10; j++ {
		for i := 0; i < 100; i++ {
			attrV := shuffle(attr, int64(i))
			copy(res, attrV)
			fmt.Println(res)
		}
		//fmt.Println(res)
	}
}

var cardColor map[int]string
func TestPoker_Shuffle(t *testing.T) {
	poker := NewPoker()

	cardColor = make(map[int]string)
	cardColor[0] = "Spade"
	cardColor[1] = "Diamond"
	cardColor[2] = "Club"
	cardColor[3] = "Heart"

	for j := 0; j < 10; j++ {
		poker.Shuffle(12345432)
		fmt.Println(poker.card)
		//for i := 0; i < len(poker.card); i++ {
		//	fmt.Printf("%s%d\n", cardColor[poker.card[i]&0xff00>>8], poker.card[i]&0xff)
		//}
	}
}

func TestPoker_Deal(t *testing.T) {
	poker := NewPoker()
	poker.Shuffle(12345432)

	for j := 0; j < 11; j++ {
		fmt.Println(poker.Deal(j))
	}
}

func TestBlue(t *testing.T) {
	temp := 0;
	r := -1;//是否有牛标志

	result := make([]int, 10)
	var offset = 0
	data := []int{10,10,10,10,10}
	//斗牛算法
	for x := 0; x < 3; x++ {
		for y := x + 1; y < 4; y++ {
			for z := y + 1; z < 5; z++ {
				if ((data[x]+data[y]+data[z])%10 == 0) {
					for j := 0; j < len(data); j++ {
						if (j != x && j != y && j != z) {
							temp += data[j];
						}
					}

					//若有牛，且剩下的两个数也是牛十
					if (temp%10 == 0) {
						r = 10;
					} else { //若有牛，剩下的不是牛十
						r = temp % 10;
					}
					result[offset] = r;
					offset++
				}
			}
		}
	}

	//没有牛
	if (r == -1) {
		fmt.Println("Failed")
		return
	}

	fmt.Println(result)

	fmt.Println(strconv.ParseInt("0x123a", 0, 64))

	addr := []string{"aaa"}
	fmt.Printf("addr:%s len:%d\n", addr, len(addr))

	addr = append(addr, "bbb")
	fmt.Printf("addr:%s len:%d\n", addr, len(addr))
}