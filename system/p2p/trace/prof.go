package trace

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const prefixPath = "/"

var profileDescriptions = map[string]string{
	"peers":        "show all dht peers which we connected",
	"blackpeers":   "show all peers from blacklist",
	"logs":         "print chain's current log,include Error,Info,Debug",
	"chainstatus":  "show this chain's current status",
	"profile":      "CPU profile. You can specify the duration in the seconds GET parameter. After you get the profile file, use the go tool pprof command to investigate the profile.",
	"threadcreate": "Stack traces that led to the creation of new OS threads",
}

type profileEntry struct {
	Name  string
	Href  string
	Desc  string
	Count int
}

func indexTmplExecute(w io.Writer, profiles []profileEntry) error {
	var b bytes.Buffer
	b.WriteString(
		fmt.Sprintf(`<html>
<head>
<title>chain33 infomathions</title>
<base  target="_blank" >

<style>
.profile-name{
	display:inline-block;
	width:6rem;
}
</style>
<!-- 每间隔5秒刷新一次页面 -->
<meta http-equiv="refresh" content="5">
<meta charset="UTF-8">
</head>
<body>
<br><h1>Chain33 FunctionList</h1>
<br>
Types of chain33 information available:
<table>
<thead><td>Count</td><td>funcName</td></thead>`))
	for index, profile := range profiles {
		link := &url.URL{Path: profile.Href}
		//fmt.Println("href:", profile.Href)
		fmt.Fprintf(&b, "<tr><td>%d</td><td><a href='%s'>%s</a></td></tr>\n", index, link, html.EscapeString(profile.Name))
	}

	b.WriteString(`</table>
<a href="goroutine">full goroutine stack dump</a>
<br/>
<p>
Funciton Descriptions:
<ul>
`)

	for _, profile := range profiles {
		fmt.Println("name:", profile.Name)
		fmt.Fprintf(&b, "<li><div class=profile-name>%s: </div> %s</li>\n", html.EscapeString(profile.Name), html.EscapeString(profile.Desc))
	}
	b.WriteString(`</ul>
</p>
</body>
</html>`)

	w.Write(b.Bytes())

	return nil
}

func Index(w http.ResponseWriter, r *http.Request) {

	if strings.HasPrefix(r.URL.Path, prefixPath) {
		name := strings.TrimPrefix(r.URL.Path, prefixPath)
		if name != "" {
			//handler(name).ServeHTTP(w, r)
			w.WriteHeader(404)
			return
		}
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var profiles []profileEntry

	var count int
	// Adding other profiles exposed from within this package
	for name, desc := range profileDescriptions {
		profiles = append(profiles, profileEntry{
			Name:  name,
			Href:  name,
			Desc:  desc,
			Count: count,
		})
		count++
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	if err := indexTmplExecute(w, profiles); err != nil {
		log.Info("Index", err)
	}
}
