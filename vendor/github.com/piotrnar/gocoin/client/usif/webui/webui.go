package webui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/piotrnar/gocoin"
	"github.com/piotrnar/gocoin/client/common"
	"github.com/piotrnar/gocoin/client/usif"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var webuimenu = [][2]string{
	{"/", "Home"},
	{"/wal", "Wallet"},
	{"/snd", "MakeTx"},
	{"/net", "Network"},
	{"/txs", "Transactions"},
	{"/blocks", "Blocks"},
	{"/miners", "Miners"},
	{"/counts", "Counters"},
}

var start_time time.Time

func ipchecker(r *http.Request) bool {
	if common.NetworkClosed || usif.Exit_now {
		return false
	}
	var a, b, c, d uint32
	n, _ := fmt.Sscanf(r.RemoteAddr, "%d.%d.%d.%d", &a, &b, &c, &d)
	if n != 4 {
		return false
	}
	addr := (a << 24) | (b << 16) | (c << 8) | d
	for i := range common.WebUIAllowed {
		if (addr & common.WebUIAllowed[i].Mask) == common.WebUIAllowed[i].Addr {
			r.ParseForm()
			return true
		}
	}
	println("ipchecker:", r.RemoteAddr, "is blocked")
	return false
}

func load_template(fn string) string {
	dat, _ := ioutil.ReadFile("www/templates/" + fn)
	return string(dat)
}

func templ_add(tmpl string, id string, val string) string {
	return strings.Replace(tmpl, id, val+id, 1)
}

func p_webui(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	pth := strings.SplitN(r.URL.Path[1:], "/", 3)
	if len(pth) == 2 {
		dat, _ := ioutil.ReadFile("www/resources/" + pth[1])
		if len(dat) > 0 {
			switch filepath.Ext(r.URL.Path) {
			case ".js":
				w.Header()["Content-Type"] = []string{"text/javascript"}
			case ".css":
				w.Header()["Content-Type"] = []string{"text/css"}
			}
			w.Write(dat)
		} else {
			http.NotFound(w, r)
		}
	}
}

func sid(r *http.Request) string {
	c, _ := r.Cookie("sid")
	if c != nil {
		return c.Value
	}
	return ""
}

func checksid(r *http.Request) bool {
	if len(r.Form["sid"]) == 0 {
		return false
	}
	if len(r.Form["sid"][0]) < 16 {
		return false
	}
	return r.Form["sid"][0] == sid(r)
}

func new_session_id(w http.ResponseWriter) (sessid string) {
	var sid [16]byte
	rand.Read(sid[:])
	sessid = hex.EncodeToString(sid[:])
	http.SetCookie(w, &http.Cookie{Name: "sid", Value: sessid})
	return
}

func write_html_head(w http.ResponseWriter, r *http.Request) {
	start_time = time.Now()

	sessid := sid(r)
	if sessid == "" {
		sessid = new_session_id(w)
	}

	s := load_template("page_head.html")
	s = strings.Replace(s, "{PAGE_TITLE}", common.CFG.WebUI.Title, 1)
	s = strings.Replace(s, "/*_SESSION_ID_*/", "var sid = '"+sessid+"'", 1)
	s = strings.Replace(s, "/*_AVERAGE_FEE_SPB_*/", fmt.Sprint("var avg_fee_spb = ", common.GetAverageFee()), 1)
	s = strings.Replace(s, "/*_SERVER_MODE_*/", fmt.Sprint("var server_mode = ", common.CFG.WebUI.ServerMode), 1)
	s = strings.Replace(s, "/*_TIME_NOW_*/", fmt.Sprint("= ", time.Now().Unix()), 1)

	if r.URL.Path != "/" {
		s = strings.Replace(s, "{HELPURL}", "help#"+r.URL.Path[1:], 1)
	} else {
		s = strings.Replace(s, "{HELPURL}", "help", 1)
	}
	s = strings.Replace(s, "{VERSION}", gocoin.Version, 1)
	if common.Testnet {
		s = strings.Replace(s, "{TESTNET}", " Testnet ", 1)
	} else {
		s = strings.Replace(s, "{TESTNET}", "", 1)
	}
	for i := range webuimenu {
		var x string
		if i > 0 && i < len(webuimenu)-1 {
			x = " | "
		}
		x += "<a "
		if r.URL.Path == webuimenu[i][0] {
			x += "class=\"menuat\" "
		}
		x += "href=\"" + webuimenu[i][0] + "\">" + webuimenu[i][1] + "</a>"
		if i == len(webuimenu)-1 {
			s = strings.Replace(s, "{MENU_LEFT}", "", 1)
			s = strings.Replace(s, "{MENU_RIGHT}", x, 1)
		} else {
			s = strings.Replace(s, "{MENU_LEFT}", x+"{MENU_LEFT}", 1)
		}
	}

	w.Write([]byte(s))
}

func write_html_tail(w http.ResponseWriter) {
	s := load_template("page_tail.html")
	s = strings.Replace(s, "<!--LOAD_TIME-->", time.Now().Sub(start_time).String(), 1)
	w.Write([]byte(s))
}

func p_help(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	fname := "help.html"
	if len(r.Form["topic"]) > 0 && len(r.Form["topic"][0]) == 4 {
		for i := 0; i < 4; i++ {
			if r.Form["topic"][0][i] < 'a' || r.Form["topic"][0][i] > 'z' {
				goto broken_topic // we only accept 4 locase characters
			}
		}
		fname = "help_" + r.Form["topic"][0] + ".html"
	}
broken_topic:

	page := load_template(fname)
	write_html_head(w, r)
	w.Write([]byte(page))
	write_html_tail(w)
}

func ServerThread(iface string) {
	http.HandleFunc("/webui/", p_webui)

	if common.FLAG.NoWallet {
		webuimenu = append(webuimenu[:1], webuimenu[3:]...) // remove Wallet and MakeTx
	} else {
		http.HandleFunc("/wal", p_wal)
		http.HandleFunc("/snd", p_snd)
		http.HandleFunc("/balance.json", json_balance)
		http.HandleFunc("/payment.zip", dl_payment)
		http.HandleFunc("/balance.zip", dl_balance)
	}

	http.HandleFunc("/net", p_net)
	http.HandleFunc("/txs", p_txs)
	http.HandleFunc("/blocks", p_blocks)
	http.HandleFunc("/miners", p_miners)
	http.HandleFunc("/counts", p_counts)
	http.HandleFunc("/cfg", p_cfg)
	http.HandleFunc("/help", p_help)

	http.HandleFunc("/txs2s.xml", xml_txs2s)
	http.HandleFunc("/txsre.xml", xml_txsre)
	http.HandleFunc("/txw4i.xml", xml_txw4i)
	http.HandleFunc("/raw_tx", raw_tx)

	http.HandleFunc("/", p_home)
	http.HandleFunc("/status.json", json_status)
	http.HandleFunc("/counts.json", json_counts)
	http.HandleFunc("/system.json", json_system)
	http.HandleFunc("/bwidth.json", json_bwidth)
	http.HandleFunc("/txstat.json", json_txstat)
	http.HandleFunc("/netcon.json", json_netcon)
	http.HandleFunc("/blocks.json", json_blocks)
	http.HandleFunc("/peerst.json", json_peerst)
	http.HandleFunc("/bwchar.json", json_bwchar)
	http.HandleFunc("/mempool_stats.json", json_mempool_stats)
	http.HandleFunc("/blkver.json", json_blkver)
	http.HandleFunc("/miners.json", json_miners)
	http.HandleFunc("/blfees.json", json_blfees)

	http.HandleFunc("/mempool_fees.txt", txt_mempool_fees)

	http.ListenAndServe(iface, nil)
}
