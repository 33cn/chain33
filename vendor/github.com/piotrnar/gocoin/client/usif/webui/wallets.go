package webui

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/piotrnar/gocoin/client/common"
	"github.com/piotrnar/gocoin/client/network"
	"github.com/piotrnar/gocoin/client/usif"
	"github.com/piotrnar/gocoin/client/wallet"
	"github.com/piotrnar/gocoin/lib/btc"
	"github.com/piotrnar/gocoin/lib/utxo"
)

type OneOut struct {
	TxId     string
	Vout     uint32
	Value    uint64
	Height   uint32
	Coinbase bool
	Message  string
	Addr     string
	Spending bool // if true the spending tx is in the mempool
}

type OneOuts struct {
	Value      uint64
	OutCnt     int
	SegWitCnt  int
	SegWitAddr string
	Outs       []OneOut

	PendingCnt   int
	PendingValue uint64
	PendingOuts  []OneOut

	SpendingValue uint64
	SpendingCnt   uint64
}

func p_wal(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}
	var str string
	common.Last.Mutex.Lock()
	if common.BlockChain.Consensus.Enforce_SEGWIT != 0 &&
		common.Last.Block.Height >= common.BlockChain.Consensus.Enforce_SEGWIT {
		str = "var segwit_active=true"
	} else {
		str = "var segwit_active=false"
	}
	common.Last.Mutex.Unlock()
	page := load_template("wallet.html")
	page = strings.Replace(page, "/*WALLET_JS_VARS*/", str, 1)
	write_html_head(w, r)
	w.Write([]byte(page))
	write_html_tail(w)
}

func json_balance(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	summary := len(r.Form["summary"]) > 0
	mempool := len(r.Form["mempool"]) > 0
	addrstring := r.Form["addrs"]
	var addrs []string
	if len(addrstring) == 0 {
		inp, err := ioutil.ReadAll(r.Body)
		if err != nil {
			println(err.Error())
			return
		}
		err = json.Unmarshal(inp, &addrs)
		if err != nil {
			println(err.Error())
			return
		}
	} else {
		addrs = strings.Split(addrstring[0], "|")
	}
	out := getUnspendlist(addrs, mempool, summary)
	bx, er := json.Marshal(out)
	if er == nil {
		w.Header()["Content-Type"] = []string{"application/json"}
		w.Write(bx)
	} else {
		println(er.Error())
	}
}

func getUnspendlist(addrs []string, mempool bool, summary bool) map[string]*OneOuts {
	out := make(map[string]*OneOuts)

	lck := new(usif.OneLock)
	lck.In.Add(1)
	lck.Out.Add(1)
	usif.LocksChan <- lck
	lck.In.Wait()

	var addr_map map[string]string

	if mempool {
		// make addrs -> idx
		addr_map = make(map[string]string, 2*len(addrs))
	}

	for _, a := range addrs {
		aa, e := btc.NewAddrFromString(a)
		if e != nil {
			continue
		}

		unsp := wallet.GetAllUnspent(aa)
		newrec := new(OneOuts)
		if len(unsp) > 0 {
			newrec.OutCnt = len(unsp)
			for _, u := range unsp {
				newrec.Value += u.Value
				network.TxMutex.Lock()
				_, spending := network.SpentOutputs[u.TxPrevOut.UIdx()]
				network.TxMutex.Unlock()
				if spending {
					newrec.SpendingValue += u.Value
					newrec.SpendingCnt++
				}
				if !summary {
					newrec.Outs = append(newrec.Outs, OneOut{
						TxId: btc.NewUint256(u.TxPrevOut.Hash[:]).String(), Vout: u.Vout,
						Value: u.Value, Height: u.MinedAt, Coinbase: u.Coinbase,
						Message: html.EscapeString(string(u.Message)), Addr: a, Spending: spending})
				}
			}
		}

		out[a] = newrec

		if mempool {
			addr_map[string(aa.OutScript())] = a
		}

		/* Segwit P2WPKH: */
		if aa.Version == btc.AddrVerPubkey(common.Testnet) {
			// SegWit if applicable
			h160 := btc.Rimp160AfterSha256(append([]byte{0, 20}, aa.Hash160[:]...))
			aa = btc.NewAddrFromHash160(h160[:], btc.AddrVerScript(common.Testnet))
			newrec.SegWitAddr = aa.String()
			unsp = wallet.GetAllUnspent(aa)
			if len(unsp) > 0 {
				newrec.OutCnt += len(unsp)
				newrec.SegWitCnt = len(unsp)
				as := aa.String()
				for _, u := range unsp {
					newrec.Value += u.Value
					if !summary {
						newrec.Outs = append(newrec.Outs, OneOut{
							TxId: btc.NewUint256(u.TxPrevOut.Hash[:]).String(), Vout: u.Vout,
							Value: u.Value, Height: u.MinedAt, Coinbase: u.Coinbase,
							Message: html.EscapeString(string(u.Message)), Addr: as})
					}
				}
			}
			if mempool {
				addr_map[string(aa.OutScript())] = a
			}
		}
	}

	// check memory pool
	if mempool {
		network.TxMutex.Lock()
		for _, t2s := range network.TransactionsToSend {
			for vo, to := range t2s.TxOut {
				if a, ok := addr_map[string(to.Pk_script)]; ok {
					newrec := out[a]
					newrec.PendingValue += to.Value
					newrec.PendingCnt++
					if !summary {
						po := &btc.TxPrevOut{Hash: t2s.Hash.Hash, Vout: uint32(vo)}
						_, spending := network.SpentOutputs[po.UIdx()]
						newrec.PendingOuts = append(newrec.PendingOuts, OneOut{
							TxId: t2s.Hash.String(), Vout: uint32(vo),
							Value: to.Value, Spending: spending})
					}
				}
			}
		}
		network.TxMutex.Unlock()
	}

	lck.Out.Done()
	return out
}

func dl_balance(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	if r.Method != "POST" {
		return
	}

	var addrs []string
	var labels []string

	if len(r.Form["addrcnt"]) != 1 {
		println("no addrcnt")
		return
	}
	addrcnt, _ := strconv.ParseUint(r.Form["addrcnt"][0], 10, 32)

	for i := 0; i < int(addrcnt); i++ {
		is := fmt.Sprint(i)
		if len(r.Form["addr"+is]) == 1 {
			addrs = append(addrs, r.Form["addr"+is][0])
			if len(r.Form["label"+is]) == 1 {
				labels = append(labels, r.Form["label"+is][0])
			} else {
				labels = append(labels, "")
			}
		}
	}

	type one_unsp_rec struct {
		btc.TxPrevOut
		Value    uint64
		Addr     string
		MinedAt  uint32
		Coinbase bool
	}

	var thisbal utxo.AllUnspentTx

	lck := new(usif.OneLock)
	lck.In.Add(1)
	lck.Out.Add(1)
	usif.LocksChan <- lck
	lck.In.Wait()

	for idx, a := range addrs {
		aa, e := btc.NewAddrFromString(a)
		aa.Extra.Label = labels[idx]
		if e == nil {
			newrecs := wallet.GetAllUnspent(aa)
			if len(newrecs) > 0 {
				thisbal = append(thisbal, newrecs...)
			}

			/* Segwit P2WPKH: */
			if aa.Version == btc.AddrVerPubkey(common.Testnet) {
				// SegWit if applicable
				h160 := btc.Rimp160AfterSha256(append([]byte{0, 20}, aa.Hash160[:]...))
				aa = btc.NewAddrFromHash160(h160[:], btc.AddrVerScript(common.Testnet))
				newrecs = wallet.GetAllUnspent(aa)
				if len(newrecs) > 0 {
					thisbal = append(thisbal, newrecs...)
				}
			}
		}
	}
	lck.Out.Done()

	buf := new(bytes.Buffer)
	zi := zip.NewWriter(buf)
	was_tx := make(map[[32]byte]bool)

	sort.Sort(thisbal)
	for i := range thisbal {
		if was_tx[thisbal[i].TxPrevOut.Hash] {
			continue
		}
		was_tx[thisbal[i].TxPrevOut.Hash] = true
		txid := btc.NewUint256(thisbal[i].TxPrevOut.Hash[:])
		fz, _ := zi.Create("balance/" + txid.String() + ".tx")
		if dat, er := common.GetRawTx(thisbal[i].MinedAt, txid); er == nil {
			fz.Write(dat)
		} else {
			println(er.Error())
		}
	}

	fz, _ := zi.Create("balance/unspent.txt")
	for i := range thisbal {
		fmt.Fprintln(fz, thisbal[i].UnspentTextLine())
	}

	zi.Close()
	w.Header()["Content-Type"] = []string{"application/zip"}
	w.Write(buf.Bytes())

}
