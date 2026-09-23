package db

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// TestGoLevelDBManifestRotationNoFileLeak 守的不变量：
//
//	MANIFEST 轮转不得让表文件（*.ldb）变成永久孤儿。
//
// 背景（issue #1396）：goleveldb 在 MANIFEST 轮转时（MaxManifestFileSize 触顶）
// 把调用方持有的 sessionRecord 直接传进 newManifest，该 record 的 addedTables
// 因此膨胀成「轮转那一刻的全量活表」。这个被污染的 record 随后被用于构造版本
// delta，refLoop 于是对这批表各多记一次永远不会配对的 +1 引用；引用计数再也回
// 不到 0，tops.remove() 永不被调用、stor.Remove 永不执行 —— 这批表文件从此
// 留在磁盘上，既不被任何版本引用（不再计入各层文件数），也不会被删除。
//
// 上游 126854af5e6d（#409）改为传 nil 修掉了它。只有「轮转那一刻已存在的表」
// 会中招，轮转之后新产生的表引用计数是干净的，所以复现必须同时做到两件事：
//
//  1. 让轮转反复发生 —— MaxManifestFileSize 压到 1 KiB；
//  2. 在轮转之后制造大规模表淘汰 —— 先铺满一整套活表，再随机顺序整片覆盖写，
//     最后全库压实一次，把「曾经活过、现在该死」的表一次性挤掉。
//
// 判据用「孤儿数相对活表数的比例」而不是绝对值：压实后的活表数会随实现细节浮动
// （实测 11 上下），绝对值阈值极易被撞穿。当前版本实测 orphans≈0（阈值 orphans
// <= live 有 ~20 倍余量），有 bug 的版本实测 orphans 是 live 的 15~25 倍。
//
// 轮转这件事本身也要被钉住：CURRENT 指向的 MANIFEST 编号在测试前后必须递增。
// 否则说明 MaxManifestFileSize 已失效（或 goleveldb 改变了轮转行为），缺陷路径根本
// 没被执行，这时孤儿数再好看也没有意义 —— 断言会直接红，而不是悄悄恒过。
//
// 本测试同时充当**依赖版本哨兵**：谁把 goleveldb 降回 64ee5596c38a 或更早，
// 这个测试就会红。
func TestGoLevelDBManifestRotationNoFileLeak(t *testing.T) {
	const (
		keyCount     = 30000   // 活表规模：太少则信号淹没在噪声里
		overwriteRds = 2       // 覆盖轮数：制造表淘汰
		valSize      = 256     // 值大小：撑大表体积，放大压实工作量
		manifestSize = 1 << 10 // 1 KiB —— 几乎每次提交都触发 MANIFEST 轮转
		writeBufSize = 256 << 10
		tableSize    = 64 << 10
	)

	dir := t.TempDir()
	d, err := leveldb.OpenFile(dir, &opt.Options{
		MaxManifestFileSize:    manifestSize,
		WriteBuffer:            writeBufSize,
		CompactionTableSize:    tableSize,
		OpenFilesCacheCapacity: 64,
	})
	require.NoError(t, err)
	defer func() { _ = d.Close() }()

	// 轮转是否真的发生，是本测试成立的前提。CURRENT 指向的 MANIFEST 编号必须在本
	// 测试期间递增；编号不变 = MaxManifestFileSize 没生效 = 缺陷路径没被执行。
	manifestBefore := manifestNum(t, dir)

	key := func(i int) []byte { return []byte(fmt.Sprintf("k%08d", i)) }
	val := bytes.Repeat([]byte("v"), valSize)

	// 阶段一：铺满一整套活表。这些表的生命周期会跨越后续每一次轮转。
	for i := 0; i < keyCount; i++ {
		if err := d.Put(key(i), val, nil); err != nil {
			t.Fatalf("阶段一写入失败 key=%d: %v", i, err)
		}
	}

	// 阶段二：随机顺序整片覆盖。活表集合在这期间不断被新表替换，
	// 而轮转一直在发生 —— 于是「被轮转钉死」和「之后被淘汰」这两件事重叠。
	rng := rand.New(rand.NewSource(1))
	for r := 0; r < overwriteRds; r++ {
		for _, i := range rng.Perm(keyCount) {
			if err := d.Put(key(i), val, nil); err != nil {
				t.Fatalf("阶段二写入失败 key=%d: %v", i, err)
			}
		}
	}

	// 阶段三：全库压实，把老表一次性淘汰掉。有 bug 时正是这一步让它们暴露成孤儿。
	if err := d.CompactRange(util.Range{}); err != nil {
		t.Fatalf("全库压实失败: %v", err)
	}

	manifestAfter := manifestNum(t, dir)
	require.Greater(t, manifestAfter, manifestBefore,
		"MANIFEST 没有发生轮转（前 %d，后 %d）：MaxManifestFileSize=%d 已失效，"+
			"本次测试没有执行到缺陷路径，结果无效",
		manifestBefore, manifestAfter, manifestSize)

	onDisk, live := awaitTableCountsStable(t, d, dir)
	orphans := onDisk - live

	t.Logf("on-disk=%d live=%d orphans=%d", onDisk, live, orphans)

	// 活表数太少说明上面的参数或 goleveldb 的属性名已经失效，测试会失去意义，
	// 这里显式挡住，避免「因为量太小所以恒过」的假绿。
	require.GreaterOrEqual(t, live, 4,
		"活表数过少（%d），无法判定；检查 leveldb.num-files-at-level{n} 属性是否仍可用", live)

	require.LessOrEqual(t, orphans, live,
		"MANIFEST 轮转后出现大量孤儿表文件（on-disk=%d live=%d orphans=%d）："+
			"轮转污染了 sessionRecord，导致表文件引用计数无法归零、stor.Remove 永不执行。"+
			"goleveldb 需 >= v1.0.1-0.20220721030215-126854af5e6d（上游 126854af / #409）",
		onDisk, live, orphans)
}

// awaitTableCountsStable 返回稳定后的（磁盘表文件数, 活表数）。
//
// goleveldb 的文件删除由后台 refLoop 异步完成，压实刚返回时磁盘上可能还短暂留着
// 几个「已不被引用、但尚未 Remove」的文件。这里轮询到孤儿数归零或不再变化为止，
// 避免把这种正常的在途文件当成泄漏误报。注意：真正被 bug 钉死的文件永远不会归零，
// 所以这个等待不会掩盖缺陷。
func awaitTableCountsStable(t *testing.T, d *leveldb.DB, dir string) (onDisk, live int) {
	t.Helper()
	const (
		pollInterval = 100 * time.Millisecond
		stableWindow = 2 * time.Second
		hardDeadline = 10 * time.Second
	)

	start := time.Now()
	last := -1
	lastChanged := time.Now()
	for {
		live = countLiveTables(t, d)
		onDisk = countTableFiles(t, dir)
		orphans := onDisk - live

		if orphans != last {
			last, lastChanged = orphans, time.Now()
		}
		// 归零即完成；或连续 stableWindow 不再变化（已排空，或确实是稳定的残留）
		if orphans <= 0 || time.Since(lastChanged) >= stableWindow {
			return onDisk, live
		}
		if time.Since(start) >= hardDeadline {
			return onDisk, live
		}
		time.Sleep(pollInterval)
	}
}

// countTableFiles 数磁盘上的表文件。goleveldb 的表文件后缀是 .ldb。
func countTableFiles(t *testing.T, dir string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.ldb"))
	require.NoError(t, err)
	return len(matches)
}

// manifestNum 返回 CURRENT 当前指向的 MANIFEST 编号（CURRENT 的内容形如
// "MANIFEST-000007"）。goleveldb 每次轮转都新建一个编号更大的 MANIFEST 并把
// CURRENT 指过去，所以这个编号单调递增 —— 它是否变大，就是「轮转有没有发生」
// 的判据。注意 CURRENT 在每次轮转时会被重写，只能在测试的稳定点读取。
func manifestNum(t *testing.T, dir string) int {
	t.Helper()
	cur, err := os.ReadFile(filepath.Join(dir, "CURRENT"))
	require.NoError(t, err, "读取 CURRENT 失败")
	name := strings.TrimSpace(string(cur))
	num, err := strconv.Atoi(strings.TrimPrefix(name, "MANIFEST-"))
	require.NoError(t, err, "解析 CURRENT 内容失败: %q", name)
	return num
}

// countLiveTables 数当前版本引用的表：各层文件数之和。
func countLiveTables(t *testing.T, d *leveldb.DB) int {
	t.Helper()
	total := 0
	for lvl := 0; lvl <= 6; lvl++ {
		v, err := d.GetProperty("leveldb.num-files-at-level" + strconv.Itoa(lvl))
		require.NoError(t, err, "读取 level%d 文件数失败", lvl)
		n, err := strconv.Atoi(v)
		require.NoError(t, err, "解析 level%d 文件数失败: %q", lvl, v)
		total += n
	}
	return total
}
