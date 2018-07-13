package types

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	randgen *rand.Rand
	randMux sync.Mutex
	Fmt          = fmt.Sprintf
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

//-------------------------------------------------------------
// clockRate is the resolution and precision of clock().
const clockRate = 20 * time.Millisecond

// czero is the process start time rounded down to the nearest clockRate
// increment.
var czero = time.Now().Round(clockRate)

// clock returns a low resolution timestamp relative to the process start time.
func clock() time.Duration {
	return time.Now().Round(clockRate).Sub(czero)
}

// clockToTime converts a clock() timestamp to an absolute time.Time value.
func clockToTime(c time.Duration) time.Time {
	return czero.Add(c)
}

// clockRound returns d rounded to the nearest clockRate increment.
func clockRound(d time.Duration) time.Duration {
	return (d + clockRate>>1) / clockRate * clockRate
}

// round returns x rounded to the nearest int64 (non-negative values only).
func round(x float64) int64 {
	if _, frac := math.Modf(x); frac >= 0.5 {
		return int64(math.Ceil(x))
	}
	return int64(math.Floor(x))
}

// Percent represents a percentage in increments of 1/1000th of a percent.
type Percent uint32

// percentOf calculates what percent of the total is x.
func percentOf(x, total float64) Percent {
	if x < 0 || total <= 0 {
		return 0
	} else if p := round(x / total * 1e5); p <= math.MaxUint32 {
		return Percent(p)
	}
	return Percent(math.MaxUint32)
}

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
