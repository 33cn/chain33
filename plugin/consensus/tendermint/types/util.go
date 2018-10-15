package types

import (
	"container/heap"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
)

const (
	RFC3339Millis = "2006-01-02T15:04:05.000Z" // forced microseconds
	timeFormat    = RFC3339Millis
)

var (
	randgen *rand.Rand
	randMux sync.Mutex
	Fmt     = fmt.Sprintf
)

func Init() {
	if randgen == nil {
		randMux.Lock()
		randgen = rand.New(rand.NewSource(time.Now().UnixNano()))
		randMux.Unlock()
	}
}

func WriteFile(filePath string, contents []byte, mode os.FileMode) error {
	return ioutil.WriteFile(filePath, contents, mode)
}

func WriteFileAtomic(filePath string, newBytes []byte, mode os.FileMode) error {
	dir := filepath.Dir(filePath)
	f, err := ioutil.TempFile(dir, "")
	if err != nil {
		return err
	}
	_, err = f.Write(newBytes)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if permErr := os.Chmod(f.Name(), mode); err == nil {
		err = permErr
	}
	if err == nil {
		err = os.Rename(f.Name(), filePath)
	}
	// any err should result in full cleanup
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

//----------------------------------------------------------
func Tempfile(prefix string) (*os.File, string) {
	file, err := ioutil.TempFile("", prefix)
	if err != nil {
		panic(err)
	}
	return file, file.Name()
}

func Fingerprint(slice []byte) []byte {
	fingerprint := make([]byte, 6)
	copy(fingerprint, slice)
	return fingerprint
}

func Kill() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGTERM)
}

func Exit(s string) {
	fmt.Printf(s + "\n")
	os.Exit(1)
}

//-----------------------------------------------------------
func Parallel(tasks ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(task func()) {
			task()
			wg.Done()
		}(task)
	}
	wg.Wait()
}

// Percent represents a percentage in increments of 1/1000th of a percent.
type Percent uint32

func (p Percent) Float() float64 {
	return float64(p) * 1e-3
}

func (p Percent) String() string {
	var buf [12]byte
	b := strconv.AppendUint(buf[:0], uint64(p)/1000, 10)
	n := len(b)
	b = strconv.AppendUint(b, 1000+uint64(p)%1000, 10)
	b[n] = '.'
	return string(append(b, '%'))
}

//-------------------------------------------------------------------
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

//--------------------------------------------------------------
func RandIntn(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	if n <= 1<<31-1 {
		randMux.Lock()
		i32 := randgen.Int31n(int32(n))
		randMux.Unlock()
		return int(i32)
	}
	randMux.Lock()
	i64 := randgen.Int63n(int64(n))
	randMux.Unlock()
	return int(i64)
}

func RandUint32() uint32 {
	randMux.Lock()
	u32 := randgen.Uint32()
	randMux.Unlock()
	return u32
}

func RandInt63n(n int64) int64 {
	randMux.Lock()
	i64 := randgen.Int63n(n)
	randMux.Unlock()
	return i64
}

//-------------------------------------------------------------
func PanicSanity(v interface{}) {
	panic(Fmt("Panicked on a Sanity Check: %v", v))
}

func PanicCrisis(v interface{}) {
	panic(Fmt("Panicked on a Crisis: %v", v))
}

func PanicQ(v interface{}) {
	panic(Fmt("Panicked questionably: %v", v))
}

//--------------------BitArray------------------------
type BitArray struct {
	*tmtypes.TendermintBitArray
	mtx sync.Mutex `json:"-"`
}

// There is no BitArray whose Size is 0.  Use nil instead.
func NewBitArray(bits int) *BitArray {
	if bits <= 0 {
		return nil
	}

	return &BitArray{TendermintBitArray: &tmtypes.TendermintBitArray{
		Bits:  int32(bits),
		Elems: make([]uint64, (bits+63)/64),
	},
	}
}

func (bA *BitArray) Size() int {
	if bA == nil {
		return 0
	}
	return int(bA.Bits)
}

// NOTE: behavior is undefined if i >= bA.Bits
func (bA *BitArray) GetIndex(i int) bool {
	if bA == nil {
		return false
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.getIndex(i)
}

func (bA *BitArray) getIndex(i int) bool {
	if i >= int(bA.Bits) {
		return false
	}
	return bA.Elems[i/64]&(uint64(1)<<uint(i%64)) > 0
}

// NOTE: behavior is undefined if i >= bA.Bits
func (bA *BitArray) SetIndex(i int, v bool) bool {
	if bA == nil {
		return false
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.setIndex(i, v)
}

func (bA *BitArray) setIndex(i int, v bool) bool {
	if i >= int(bA.Bits) {
		return false
	}
	if v {
		bA.Elems[i/64] |= (uint64(1) << uint(i%64))
	} else {
		bA.Elems[i/64] &= ^(uint64(1) << uint(i%64))
	}
	return true
}

func (bA *BitArray) Copy() *BitArray {
	if bA == nil {
		return nil
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.copy()
}

func (bA *BitArray) copy() *BitArray {
	c := make([]uint64, len(bA.Elems))
	copy(c, bA.Elems)
	return &BitArray{TendermintBitArray: &tmtypes.TendermintBitArray{
		Bits:  bA.Bits,
		Elems: c,
	},
	}
}

func (bA *BitArray) copyBits(bits int) *BitArray {
	c := make([]uint64, (bits+63)/64)
	copy(c, bA.Elems)
	return &BitArray{TendermintBitArray: &tmtypes.TendermintBitArray{
		Bits:  int32(bits),
		Elems: c,
	},
	}
}

// Returns a BitArray of larger bits size.
func (bA *BitArray) Or(o *BitArray) *BitArray {
	if bA == nil && o.TendermintBitArray == nil {
		return nil
	}
	if bA == nil && o != nil {
		o.Copy()
	}
	if o.TendermintBitArray == nil {
		return bA.Copy()
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	c := bA.copyBits(MaxInt(int(bA.Bits), int(o.Bits)))
	for i := 0; i < len(c.Elems); i++ {
		c.Elems[i] |= o.Elems[i]
	}
	return c
}

// Returns a BitArray of smaller bit size.
func (bA *BitArray) And(o *BitArray) *BitArray {
	if bA == nil || o.TendermintBitArray == nil {
		return nil
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.and(o)
}

func (bA *BitArray) and(o *BitArray) *BitArray {
	c := bA.copyBits(MinInt(int(bA.Bits), int(o.Bits)))
	for i := 0; i < len(c.Elems); i++ {
		c.Elems[i] &= o.Elems[i]
	}
	return c
}

func (bA *BitArray) Not() *BitArray {
	if bA == nil {
		return nil // Degenerate
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	c := bA.copy()
	for i := 0; i < len(c.Elems); i++ {
		c.Elems[i] = ^c.Elems[i]
	}
	return c
}

func (bA *BitArray) Sub(o *BitArray) *BitArray {
	if bA == nil || o.TendermintBitArray == nil {
		return nil
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	if bA.Bits > o.Bits {
		c := bA.copy()
		for i := 0; i < len(o.Elems)-1; i++ {
			c.Elems[i] &= ^c.Elems[i]
		}
		i := len(o.Elems) - 1
		if i >= 0 {
			for idx := i * 64; idx < int(o.Bits); idx++ {
				// NOTE: each individual GetIndex() call to o is safe.
				c.setIndex(idx, c.getIndex(idx) && !o.GetIndex(idx))
			}
		}
		return c
	}
	return bA.and(o.Not()) // Note degenerate case where o == nil
}

func (bA *BitArray) IsEmpty() bool {
	if bA == nil {
		return true // should this be opposite?
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	for _, e := range bA.Elems {
		if e > 0 {
			return false
		}
	}
	return true
}

func (bA *BitArray) IsFull() bool {
	if bA == nil {
		return true
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()

	// Check all elements except the last
	for _, elem := range bA.Elems[:len(bA.Elems)-1] {
		if (^elem) != 0 {
			return false
		}
	}

	// Check that the last element has (lastElemBits) 1's
	lastElemBits := (bA.Bits+63)%64 + 1
	lastElem := bA.Elems[len(bA.Elems)-1]
	return (lastElem+1)&((uint64(1)<<uint(lastElemBits))-1) == 0
}

func (bA *BitArray) PickRandom() (int, bool) {
	if bA == nil {
		return 0, false
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()

	length := len(bA.Elems)
	if length == 0 {
		return 0, false
	}
	randElemStart := RandIntn(length)
	for i := 0; i < length; i++ {
		elemIdx := (i + randElemStart) % length
		if elemIdx < length-1 {
			if bA.Elems[elemIdx] > 0 {
				randBitStart := RandIntn(64)
				for j := 0; j < 64; j++ {
					bitIdx := (j + randBitStart) % 64
					if (bA.Elems[elemIdx] & (uint64(1) << uint(bitIdx))) > 0 {
						return 64*elemIdx + bitIdx, true
					}
				}
				panic("should not happen")
			}
		} else {
			// Special case for last elem, to ignore straggler bits
			elemBits := int(bA.Bits) % 64
			if elemBits == 0 {
				elemBits = 64
			}
			randBitStart := RandIntn(elemBits)
			for j := 0; j < elemBits; j++ {
				bitIdx := ((j + randBitStart) % elemBits)
				if (bA.Elems[elemIdx] & (uint64(1) << uint(bitIdx))) > 0 {
					return 64*elemIdx + bitIdx, true
				}
			}
		}
	}
	return 0, false
}

func (bA *BitArray) String() string {
	if bA == nil {
		return "nil-BitArray"
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.stringIndented("")
}

func (bA *BitArray) StringIndented(indent string) string {
	if bA == nil {
		return "nil-BitArray"
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	return bA.stringIndented(indent)
}

func (bA *BitArray) stringIndented(indent string) string {

	lines := []string{}
	bits := ""
	for i := 0; i < int(bA.Bits); i++ {
		if bA.getIndex(i) {
			bits += "X"
		} else {
			bits += "_"
		}
		if i%100 == 99 {
			lines = append(lines, bits)
			bits = ""
		}
		if i%10 == 9 {
			bits += " "
		}
		if i%50 == 49 {
			bits += " "
		}
	}
	if len(bits) > 0 {
		lines = append(lines, bits)
	}
	return fmt.Sprintf("BA{%v:%v}", bA.Bits, strings.Join(lines, indent))
}

func (bA *BitArray) Bytes() []byte {
	bA.mtx.Lock()
	defer bA.mtx.Unlock()

	numBytes := (bA.Bits + 7) / 8
	bytes := make([]byte, numBytes)
	for i := 0; i < len(bA.Elems); i++ {
		elemBytes := [8]byte{}
		binary.LittleEndian.PutUint64(elemBytes[:], bA.Elems[i])
		copy(bytes[i*8:], elemBytes[:])
	}
	return bytes
}

// NOTE: other bitarray o is not locked when reading,
// so if necessary, caller must copy or lock o prior to calling Update.
// If bA is nil, does nothing.
func (bA *BitArray) Update(o *BitArray) {
	if bA == nil || o.TendermintBitArray == nil {
		return
	}
	bA.mtx.Lock()
	defer bA.mtx.Unlock()
	copy(bA.Elems, o.Elems)
}

//------------------Heap----------------------
type Comparable interface {
	Less(o interface{}) bool
}

//-----------------------------------------------------------------------------

/*
Example usage:
	h := NewHeap()

	h.Push(String("msg1"), 1)
	h.Push(String("msg3"), 3)
	h.Push(String("msg2"), 2)

	fmt.Println(h.Pop())
	fmt.Println(h.Pop())
	fmt.Println(h.Pop())
*/

type Heap struct {
	pq priorityQueue
}

func NewHeap() *Heap {
	return &Heap{pq: make([]*pqItem, 0)}
}

func (h *Heap) Len() int64 {
	return int64(len(h.pq))
}

func (h *Heap) Push(value interface{}, priority Comparable) {
	heap.Push(&h.pq, &pqItem{value: value, priority: priority})
}

func (h *Heap) Peek() interface{} {
	if len(h.pq) == 0 {
		return nil
	}
	return h.pq[0].value
}

func (h *Heap) Update(value interface{}, priority Comparable) {
	h.pq.Update(h.pq[0], value, priority)
}

func (h *Heap) Pop() interface{} {
	item := heap.Pop(&h.pq).(*pqItem)
	return item.value
}

//-----------------------------------------------------------------------------

///////////////////////
// From: http://golang.org/pkg/container/heap/#example__priorityQueue

type pqItem struct {
	value    interface{}
	priority Comparable
	index    int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].priority.Less(pq[j].priority)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*pqItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *priorityQueue) Update(item *pqItem, value interface{}, priority Comparable) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}
