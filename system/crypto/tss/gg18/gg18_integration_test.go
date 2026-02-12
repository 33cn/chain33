package gg18

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
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
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/signer"
	"github.com/stretchr/testify/require"
)

const (
	tssThreshold = 3
	testChannel  = int32(20260201)
	tssMessage   = "gg18-integration-test"
)

func TestGG18_4Node(t *testing.T) {

	channel := testChannel
	ports := make([]int, 4)
	for i := range ports {
		ports[i] = getRandomPort(t)
	}
	log.Info("TestGG18Integration4Node", "ports", ports, "channel", channel, "name", t.Name())
	mock1, cli1 := startTestNode(t, ports[0], channel, nil)
	defer mock1.Close()

	selfID := waitSelfPeerID(t, cli1, 10*time.Second)
	seed := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", ports[0], selfID)

	// rank 0, 1,1,1
	for i := 1; i <= 3; i++ {
		role := fmt.Sprintf("node%d", i+1)
		cmd := startChildNode(t, role, ports[i], seed)
		defer waitChildExit(t, cmd, role)
	}

	runNodeFlow(t, cli1, 0, "node1")
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

func TestGG18Node(t *testing.T) {
	if testing.Short() {
		t.Skip("skip gg18 integration test in short mode")
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

	mock, cli := startTestNode(t, port, testChannel, []string{seed})
	defer mock.Close()

	runNodeFlow(t, cli, 1, role)
}

func runNodeFlow(t *testing.T, cli queue.Client, rank uint32, role string) {
	peers := waitPeerIDs(t, cli, 4, 30*time.Second, role)
	log.Info("runNodeFlow dkg start", "role", role)
	dkgRes, err := ProcessDKG(peers, tssThreshold, rank, "dkg-session-id")
	require.NoError(t, err)
	msg := []byte(tssMessage)
	log.Info("runNodeFlow sign start", "role", role)
	signRes, err := ProcessSign(peers, msg, dkgRes, tssThreshold, "sign-session")
	require.NoError(t, err)
	pubKey, err := tss.ParseBtcecPublicKey(dkgRes)
	require.NoError(t, err)
	verifySignatureWithDKG(t, pubKey, msg, signRes)
	log.Info("runNodeFlow reshare start", "role", role)
	reshareRes, err := ProcessReshare(peers, dkgRes, tssThreshold, "reshare-session-id")
	require.NoError(t, err)
	require.NotNil(t, reshareRes)
	if role == "node4" {
		log.Info("node4 exit test", "peers", peers, "role", role)
		return
	}
	// test 3 node sign
	dkgRes.Share = reshareRes.Share.Bytes()
	peers = waitPeerIDs(t, cli, 3, 30*time.Second, role)
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			id := fmt.Sprintf("sign-session-%d", idx)
			signMsg := []byte(id)
			log.Info("test 3 node concurrent sign start", "role", role, "id", id)
			signRes, err := ProcessSign(peers, signMsg, dkgRes, tssThreshold, id)
			require.NoError(t, err)
			verifySignatureWithDKG(t, pubKey, signMsg, signRes)
			log.Info("test 3 node concurrent sign end", "role", role, "id", id)
			wg.Done()
		}(i + 1)
	}
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()
	select {
	case <-c:
	case <-time.After(30 * time.Second):
		require.FailNow(t, "test 3 node concurrent sign timeout")
	}
	return

}

func verifySignatureWithDKG(t *testing.T, pubKey *btcec.PublicKey, msg []byte, signRes *signer.Result) {
	sig, err := AliceToBtcecSignature(signRes)
	require.NoError(t, err)
	ok := sig.Verify(msg, pubKey)
	require.True(t, ok, "signature must verify with DKG public key")
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
	cmd *exec.Cmd
	out *bytes.Buffer
}

func startChildNode(t *testing.T, role string, port int, seed string) *childProc {
	testName := "TestGG18Node"
	args := []string{"test", "./", "-run", "^" + testName + "$", "-count=1"}
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "TSS_ROLE="+role,
		"TSS_PORT="+strconv.Itoa(port), "TSS_SEEDS="+seed,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
	})
	return &childProc{cmd: cmd, out: &out}
}

func waitChildExit(t *testing.T, child *childProc, label string) {
	err := child.cmd.Wait()
	if err != nil {
		t.Fatalf("%s process failed: %v\n%s", label, err, child.out.String())
	}
}

func waitPeerIDs(t *testing.T, cli queue.Client, want int, timeout time.Duration, role string) []string {
	deadline := time.Now().Add(timeout)
	peers, err := tss.FetchConnectedPeers(cli, 3*time.Second)
	for time.Now().Before(deadline) {
		if err == nil && len(peers) == want {
			ids := make([]string, 0, len(peers))
			for _, peer := range peers {
				ids = append(ids, peer.Name)
			}
			return ids
		}
		time.Sleep(time.Second)
		peers, err = tss.FetchConnectedPeers(cli, 3*time.Second)
	}
	t.Fatalf("timeout waiting for %d peers, actual=%d, role %s", want, len(peers), role)
	return nil
}

func waitSelfPeerID(t *testing.T, cli queue.Client, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		peers, err := tss.FetchConnectedPeers(cli, 3*time.Second)
		if err == nil && len(peers) > 0 {
			return peers[len(peers)-1].Name
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("timeout waiting for self peer id")
	return ""
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
