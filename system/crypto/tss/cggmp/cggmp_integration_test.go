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
	"github.com/stretchr/testify/require"
)

const (
	cggmpThreshold = 3
	testChannel    = int32(20260202)
	tssMessage     = "cggmp-integration-test"

	dkgSessionID     = "cggmp-dkg-session-id"
	refreshSessionID = "cggmp-refresh-session-id"
	signSessionID    = "cggmp-sign-session"

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
//	3-node subset: threshold sign + concurrent signs, every signature verified with the DKG group key.
//
// The refresh phase is what CGGMP adds over GG18 and is only meaningful across processes
// (it depends on the shared ssid derived from the DKG rid), so it is exercised here end to
// end: every node must derive the same ssid or the zero-knowledge challenges fail.
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
	runNodeFlow(t, cli1, 0, "node1")
	signalBarrier(barrierDir, "node1")
	waitBarrier(t, barrierDir, barrierTimeout)

	wg := sync.WaitGroup{}
	for _, cmd := range cmds {
		wg.Add(1)
		go func(cmd *childProc) {
			waitChildExit(t, cmd, cmd.role)
			wg.Done()
		}(cmd)
	}
	wg.Wait()
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

	runNodeFlow(t, cli, 1, role)
	if barrierDir != "" && role != "node4" {
		signalBarrier(barrierDir, role)
		waitBarrier(t, barrierDir, barrierTimeout)
	}
}

// runNodeFlow drives one node through every CGGMP phase. All four nodes run DKG and refresh
// together; node4 then leaves the test, and the remaining three sign the same key with
// threshold 3. rank 0/1 keeps the Birkhoff parameters of the 3-node subset valid
// (rank <= threshold-2 = 1 with threshold 3).
func runNodeFlow(t *testing.T, cli queue.Client, rank uint32, role string) {
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
	signRes, err := ProcessSign(peers, cggmpThreshold, msg, dkgRes, refreshRes, signSessionID, WithTimeout(signOpTimeout))
	require.NoError(t, err)
	verifySignatureWithDKG(t, pubKey, msg, signRes)
	log.Info("runNodeFlow subset sign end", "role", role)

	// 并发发起多路签名:不设墙钟「预算」闸,只要全部能完成即通过,算得慢也不判失败。
	// 正确性由每路 goroutine 内的 verifySignatureWithDKG 验签独立保证;
	// 单路真正卡死由 ProcessSign 的兜底超时(signOpTimeout)中断,最终由 go test 包级超时兜底。
	wg := sync.WaitGroup{}
	for i := 0; i < concurrentSigns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("%s-%d", signSessionID, idx)
			signMsg := []byte(id)
			log.Info("concurrent sign start", "role", role, "id", id)
			res, err := ProcessSign(peers, cggmpThreshold, signMsg, dkgRes, refreshRes, id, WithTimeout(signOpTimeout))
			require.NoError(t, err)
			verifySignatureWithDKG(t, pubKey, signMsg, res)
			log.Info("concurrent sign end", "role", role, "id", id)
		}(i + 1)
	}
	wg.Wait()
}

func verifySignatureWithDKG(t *testing.T, pubKey *btcec.PublicKey, msg []byte, signRes *alicesign.Result) {
	sig, err := ToBtcecSignature(signRes)
	require.NoError(t, err)
	ok := sig.Verify(msg, pubKey)
	require.True(t, ok, "signature must verify with DKG public key")
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
	out  *bytes.Buffer
	role string
}

func startChildNode(t *testing.T, role string, port int, seed, barrierDir string) *childProc {

	cmd := exec.Command(os.Args[0], "-test.run", "^TestCGGMPNode$", "-test.v")
	cmd.Env = append(os.Environ(), "TSS_ROLE="+role,
		"TSS_PORT="+strconv.Itoa(port), "TSS_SEED="+seed,
		"TSS_BARRIER_DIR="+barrierDir,
	)
	var out bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &out)
	cmd.Stderr = io.MultiWriter(os.Stderr, &out)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
	})
	return &childProc{cmd: cmd, out: &out, role: role}
}

func waitChildExit(t *testing.T, child *childProc, role string) {
	log.Info("waitChildExit", "role", role)
	c := make(chan error)
	go func() {
		c <- child.cmd.Wait()
	}()
	select {
	case err := <-c:
		if err != nil {
			t.Fatalf("child exit with error: %v, role %s, output: %s", err, role, child.out.String())
		}
	case <-time.After(childExitTimeout):
		t.Fatalf("timeout waiting for child exit, role %s, output: %s", role, child.out.String())
	}
	log.Info("waitChildExit end", "role", role)
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
