// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/system/crypto/tss"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/btcsuite/btcd/btcec/v2"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cggmpThreshold = 3
	testChannel    = int32(20260202)
	tssMessage     = "cggmp-integration-test"

	dkgSessionID          = "cggmp-dkg-session-id"
	refreshSessionID      = "cggmp-refresh-session-id"
	refreshAgainSessionID = "cggmp-refresh-again-session-id"
	signSessionID         = "cggmp-sign-session"

	// 以下超时均为「防止真正卡死」的兜底值,远大于正常(含 -race)耗时,
	// 正常或偏慢都不应触发;用例是否通过由密码学验签(verifySignatureWithDKG)判定,
	// 真正卡死时由 go test 默认包级超时(10min)兜底。
	peerWait4Timeout = 150 * time.Second // 等齐 4 个节点(含自身)的发现超时
	peerWait3Timeout = 90 * time.Second  // 等齐 3 个签名节点的发现超时
	dkgOpTimeout     = 2 * time.Minute   // DKG(+partial public key 交换)兜底超时
	// refresh 每个节点都要生成一个 2048-bit Paillier 密钥(安全素数),是三个阶段里最慢的,
	// 比 GG18 的 reshare 重得多,故给到 5min。
	refreshOpTimeout = 5 * time.Minute   // refresh 兜底超时
	signOpTimeout    = 2 * time.Minute   // 单次签名兜底超时
	barrierTimeout   = 120 * time.Second // 进程间 barrier 同步超时
	childExitTimeout = 6 * time.Minute   // 等待子进程退出超时
	concurrentSigns  = 4                 // 并发签名路数
)

// TestCGGMP4Node runs the CGGMP wrapper over four real chain33 test nodes (one in-process,
// three child processes) connected through the real p2p + queue transport:
//
//	all 4 nodes: DKG(threshold 3, includes the partial public key exchange)
//	             -> refresh (key refresh + Paillier/Pedersen provisioning)
//	             -> a second refresh through ProcessRefreshAgain (all 4 nodes still present)
//	3-node subset: threshold sign + concurrent signs, every signature verified with the DKG group key.
//
// The refresh phase is what CGGMP adds over GG18 and is only meaningful across processes
// (it depends on the shared ssid derived from the DKG rid), so it is exercised here end to
// end: every node must derive the same ssid or the zero-knowledge challenges fail. The second
// refresh proves the public ProcessRefreshAgain entry point rotates a generation that can still
// sign under the unchanged DKG group key; the four nodes' ssids are compared across processes.
func TestCGGMP4Node(t *testing.T) {

	if testing.Short() {
		t.Skip("skip cggmp integration test in short mode")
	}
	channel := testChannel
	ports := make([]int, 4)
	for i := range ports {
		ports[i] = getRandomPort(t)
	}

	mock1, cli1 := startTestNode(t, ports[0], channel, nil)
	defer mock1.Close()

	selfID := waitSelfPeerID(t, cli1, 10*time.Second)
	seed := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", ports[0], selfID)
	log.Info("TestCGGMP4Node", "ports", ports, "channel", channel, "seed", seed)

	barrierDir := t.TempDir()
	cmds := make([]*childProc, 0, 3)
	for i := 1; i <= 3; i++ {
		role := fmt.Sprintf("node%d", i+1)
		log.Info("TestCGGMP4Node start child node", "role", role, "port", ports[i])
		cmds = append(cmds, startChildNode(t, role, ports[i], seed, barrierDir))
	}
	runNodeFlow(t, cli1, 0, "node1", barrierDir)
	signalBarrier(barrierDir, "node1")
	waitBarrier(t, barrierDir, barrierTimeout)
	// Every node published the ssid of the second refresh; they must be identical because the ssid
	// is the zero-knowledge domain separator, and a disagreement silently breaks the proofs.
	assertSameSSID(t, barrierDir)

	wg := sync.WaitGroup{}
	errCh := make(chan error, len(cmds))
	for _, cmd := range cmds {
		wg.Add(1)
		go func(cmd *childProc) {
			defer wg.Done()
			if err := waitChildExit(cmd, cmd.role); err != nil {
				errCh <- err
			}
		}(cmd)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var firstErr error
	select {
	case <-done:
	case firstErr = <-errCh:
		// A child failed: stop the rest so the test returns without waiting out their
		// individual exit timeouts.
		for _, cmd := range cmds {
			if cmd.cmd.Process != nil {
				_ = cmd.cmd.Process.Kill()
			}
		}
		<-done
	}
	close(errCh)
	for err := range errCh {
		if firstErr == nil {
			firstErr = err
		}
	}
	require.NoError(t, firstErr)
}

// TestCGGMPNode is the child-process entry point; it is a no-op unless TSS_ROLE is set by
// TestCGGMP4Node (which re-executes this test binary).
func TestCGGMPNode(t *testing.T) {
	if testing.Short() {
		t.Skip("skip cggmp integration test in short mode")
	}
	role := strings.TrimSpace(os.Getenv("TSS_ROLE"))
	if !strings.Contains(role, "node") {
		t.Skip("not nodeX process")
	}
	runChildNode(t, role)
}

func runChildNode(t *testing.T, role string) {
	port := getEnvInt(t, "TSS_PORT")
	seed := strings.TrimSpace(os.Getenv("TSS_SEED"))
	barrierDir := strings.TrimSpace(os.Getenv("TSS_BARRIER_DIR"))
	log.Info("runChildNode", "role", role, "port", port, "seed", seed)
	mock, cli := startTestNode(t, port, testChannel, []string{seed})
	defer mock.Close()

	runNodeFlow(t, cli, 1, role, barrierDir)
	if barrierDir != "" && role != "node4" {
		signalBarrier(barrierDir, role)
		waitBarrier(t, barrierDir, barrierTimeout)
	}
}

// runNodeFlow drives one node through every CGGMP phase. All four nodes run DKG and two refreshes
// together; node4 then leaves the test, and the remaining three sign the same key with threshold 3
// using the second refresh generation. rank 0/1 keeps the Birkhoff parameters of the 3-node subset
// valid (rank <= threshold-2 = 1 with threshold 3).
func runNodeFlow(t *testing.T, cli queue.Client, rank uint32, role, barrierDir string) {
	peers := waitPeerIDs(t, cli, 4, peerWait4Timeout, role)
	log.Info("runNodeFlow dkg start", "role", role)
	dkgRes, err := ProcessDKG(peers, cggmpThreshold, rank, dkgSessionID, WithTimeout(dkgOpTimeout))
	require.NoError(t, err)
	// ProcessDKG already runs the partial public key exchange (phase 2) internally; make sure
	// every participant's g^{share} was collected, refresh needs the full set.
	require.Len(t, dkgRes.PartialPubKeys, len(peers), "partial public keys must be collected from every peer")
	require.NotEmpty(t, dkgRes.Rid)

	log.Info("runNodeFlow refresh start", "role", role)
	refreshRes, err := ProcessRefresh(peers, cggmpThreshold, dkgRes, refreshSessionID, WithTimeout(refreshOpTimeout))
	require.NoError(t, err)
	require.NotNil(t, refreshRes)
	// The signer subset must be able to find every signer's partial public key and Pedersen
	// parameters in the refresh output, so all participants must be present here.
	require.Len(t, refreshRes.PartialPubKeys, len(peers), "refresh must cover every participant")
	require.Len(t, refreshRes.PedParams, len(peers), "refresh must provision every participant")

	// Second refresh through the public API, with all four nodes still connected: this rotates the
	// generation and is what ProcessSign below consumes, so the integration test covers
	// ProcessRefreshAgain end to end (not only the in-process helper path).
	log.Info("runNodeFlow second refresh start", "role", role)
	refreshAgainRes, err := ProcessRefreshAgain(peers, cggmpThreshold, dkgRes, refreshRes,
		refreshAgainSessionID, WithTimeout(refreshOpTimeout))
	require.NoError(t, err)
	require.NotNil(t, refreshAgainRes)
	require.NotEqual(t, refreshRes.Share, refreshAgainRes.Share, "the second refresh must rotate the share")
	require.Len(t, refreshAgainRes.PartialPubKeys, len(peers), "second refresh must cover every participant")
	require.Len(t, refreshAgainRes.PedParams, len(peers), "second refresh must provision every participant")
	// Every node records its ssid; the parent compares them once all four have finished.
	publishSSID(barrierDir, role, refreshAgainRes.Ssid)
	log.Info("runNodeFlow second refresh end", "role", role)

	if role == "node4" {
		log.Info("node4 exit test", "peers", peers, "role", role)
		return
	}

	// node4 left: wait until exactly the 3 signing nodes are connected, then run the threshold
	// sign with that subset. The same peers list is used on all signing nodes; only those
	// signers' bks / partial public keys / Pedersen parameters are fed to the sign core.
	peers = waitPeerIDs(t, cli, cggmpThreshold, peerWait3Timeout, role)
	pubKey := dkgResultPublicKey(t, dkgRes)

	msg := []byte(tssMessage)
	log.Info("runNodeFlow subset sign start", "role", role, "peers", peers)
	signRes, err := ProcessSign(peers, cggmpThreshold, msg, dkgRes, refreshAgainRes, signSessionID, WithTimeout(signOpTimeout))
	require.NoError(t, err)
	verifySignatureWithDKG(t, pubKey, msg, signRes)
	log.Info("runNodeFlow subset sign end", "role", role)

	// 并发发起多路签名:不设墙钟「预算」闸,只要全部能完成即通过,算得慢也不判失败。
	// 正确性由每路 goroutine 内的 verifySignature 验签独立保证;
	// 单路真正卡死由 ProcessSign 的兜底超时(signOpTimeout)中断,最终由 go test 包级超时兜底。
	// 失败经 error channel 回到测试 goroutine 再用 assert 报告:t.FailNow (require) 只能在测试
	// goroutine 调用,在子 goroutine 里调用只会退出那个 goroutine 并留下 WaitGroup 悬挂。
	signErrCh := make(chan error, concurrentSigns)
	wg := sync.WaitGroup{}
	for i := 0; i < concurrentSigns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("%s-%d", signSessionID, idx)
			signMsg := []byte(id)
			log.Info("concurrent sign start", "role", role, "id", id)
			res, err := ProcessSign(peers, cggmpThreshold, signMsg, dkgRes, refreshAgainRes, id, WithTimeout(signOpTimeout))
			if err != nil {
				signErrCh <- fmt.Errorf("concurrent sign %s: %w", id, err)
				return
			}
			signErrCh <- verifySignature(pubKey, signMsg, res)
			log.Info("concurrent sign end", "role", role, "id", id)
		}(i + 1)
	}
	wg.Wait()
	close(signErrCh)
	for err := range signErrCh {
		assert.NoError(t, err)
	}
}

func verifySignatureWithDKG(t *testing.T, pubKey *btcec.PublicKey, msg []byte, signRes *alicesign.Result) {
	t.Helper()
	require.NoError(t, verifySignature(pubKey, msg, signRes))
}

// verifySignature converts the alice result and checks it against the DKG group public key,
// returning an error instead of failing the test so goroutines can report through a channel.
func verifySignature(pubKey *btcec.PublicKey, msg []byte, signRes *alicesign.Result) error {
	sig, err := ToBtcecSignature(signRes)
	if err != nil {
		return err
	}
	if !sig.Verify(msg, pubKey) {
		return fmt.Errorf("signature does not verify with the DKG public key")
	}
	return nil
}

func getRandomPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	port := addr.Port
	_ = l.Close()
	return port
}

func startTestNode(t *testing.T, port int, channel int32, seeds []string) (*testnode.Chain33Mock, queue.Client) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.GetModuleConfig().P2P.Enable = true
	cfg.GetModuleConfig().P2P.WaitPid = true
	cfg.GetModuleConfig().P2P.Types = []string{p2pty.DHTTypeName}
	cfg.GetModuleConfig().Crypto.EnableTSS = true
	cfg.GetModuleConfig().Log.LogFile = ""

	subCfg := &p2pty.P2PSubConfig{
		Port:    int32(port),
		Channel: channel,
		Seeds:   seeds,
	}
	// NAT port mapping is meaningless for this test, and go-libp2p v0.35.0's p2p/net/nat/nat.go
	// has a data race (extAddr is written by the mapping refresh without the lock that GetMapping
	// reads it under), which the -race integration job flags on hosts that can discover a NAT.
	// Disable it for every node — the in-process parent and the child processes — through
	// [p2p].isNatPortMap.
	subCfg.SetNatPortMap(false)
	jcfg, err := json.Marshal(subCfg)
	require.NoError(t, err)
	if cfg.GetSubConfig().P2P == nil {
		cfg.GetSubConfig().P2P = make(map[string][]byte)
	}
	cfg.GetSubConfig().P2P[p2pty.DHTTypeName] = jcfg

	mock := testnode.NewWithConfig(cfg, nil)
	return mock, mock.GetClient()
}

type childProc struct {
	cmd  *exec.Cmd
	out  *syncBuffer
	role string
}

// syncBuffer is a bytes.Buffer guarded by a mutex. os/exec copies the child's stdout and stderr on
// two separate goroutines, and both are wired to the same destination: a bare bytes.Buffer races
// under -race.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func startChildNode(t *testing.T, role string, port int, seed, barrierDir string) *childProc {

	cmd := exec.Command(os.Args[0], "-test.run", "^TestCGGMPNode$", "-test.v")
	cmd.Env = append(os.Environ(), "TSS_ROLE="+role,
		"TSS_PORT="+strconv.Itoa(port), "TSS_SEED="+seed,
		"TSS_BARRIER_DIR="+barrierDir,
	)
	out := &syncBuffer{}
	cmd.Stdout = io.MultiWriter(os.Stdout, out)
	cmd.Stderr = io.MultiWriter(os.Stderr, out)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
	})
	return &childProc{cmd: cmd, out: out, role: role}
}

// waitChildExit waits for a child process and returns an error instead of calling t.Fatalf, so it
// can run on a helper goroutine (t.Fatalf there only exits that goroutine and would leave the
// WaitGroup waiting until the package timeout).
func waitChildExit(child *childProc, role string) error {
	log.Info("waitChildExit", "role", role)
	c := make(chan error, 1)
	go func() {
		c <- child.cmd.Wait()
	}()
	select {
	case err := <-c:
		if err != nil {
			return fmt.Errorf("child exit with error: %v, role %s, output: %s", err, role, child.out.String())
		}
	case <-time.After(childExitTimeout):
		return fmt.Errorf("timeout waiting for child exit, role %s, output: %s", role, child.out.String())
	}
	log.Info("waitChildExit end", "role", role)
	return nil
}

func waitPeerIDs(t *testing.T, cli queue.Client, want int, timeout time.Duration, role string) []string {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		peers, err := tss.FetchConnectedPeers(cli, 3*time.Second)
		if err == nil && len(peers) == want {
			ids := make([]string, 0, len(peers))
			for _, peer := range peers {
				ids = append(ids, peer.Name)
			}
			return ids
		}
		log.Info("waitPeerIDs", "role", role, "want", want, "actual", len(peers), "err", err)
		time.Sleep(time.Second * 3)
	}
	t.Fatalf("timeout waiting for %d peers, role %s", want, role)
	return nil
}

func waitSelfPeerID(t *testing.T, cli queue.Client, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		peers, err := tss.FetchConnectedPeers(cli, 3*time.Second)
		if err == nil && len(peers) > 0 && peers[len(peers)-1].Name != "" {
			return peers[len(peers)-1].Name
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("timeout waiting for self peer id")
	return ""
}

var barrierPeers = []string{"node1", "node2", "node3"}

func signalBarrier(dir, role string) {
	_ = os.WriteFile(filepath.Join(dir, role+".done"), []byte("ok"), 0644)
}

// publishSSID records this node's second-refresh ssid so the parent process can compare all four
// nodes' values. The ssid is derived independently per node; a disagreement would only surface as
// an opaque zero-knowledge failure, so the test pins it explicitly.
func publishSSID(dir, role string, ssid []byte) {
	if dir == "" {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "ssid-"+role+".bin"), ssid, 0644)
}

// assertSameSSID asserts the four nodes derived the same second-refresh ssid. It must run after the
// barrier, by which point every child has published its ssid.
func assertSameSSID(t *testing.T, dir string) {
	t.Helper()
	var want []byte
	for _, role := range []string{"node1", "node2", "node3", "node4"} {
		got, err := os.ReadFile(filepath.Join(dir, "ssid-"+role+".bin"))
		require.NoError(t, err, "missing ssid from %s", role)
		if want == nil {
			want = got
			continue
		}
		require.True(t, bytes.Equal(want, got), "ssid mismatch: node1=%x %s=%x", want, role, got)
	}
	require.NotEmpty(t, want, "refresh ssid must not be empty")
}

func waitBarrier(t *testing.T, dir string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allDone := true
		for _, role := range barrierPeers {
			if _, err := os.Stat(filepath.Join(dir, role+".done")); err != nil {
				allDone = false
				break
			}
		}
		if allDone {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("waitBarrier timeout waiting for peers: %v", barrierPeers)
}

func getEnvInt(t *testing.T, key string) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		t.Fatalf("missing env %s", key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("invalid env %s=%q", key, v)
	}
	return n
}
