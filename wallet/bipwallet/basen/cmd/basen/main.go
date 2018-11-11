// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/33cn/chain33/wallet/bipwallet/basen"
)

var decoder = flag.String("decode", "none", "input decoding method")
var encoder = flag.String("encode", "none", "output encoding method")
var help = flag.Bool("help", false, "show help")

type codec struct {
	label          string
	raw            bool
	EncodeToString func([]byte) string
	DecodeString   func(string) ([]byte, error)
}

var codecs = map[string]codec{
	"none": {
		label:          "No encoding, raw binary",
		raw:            true,
		EncodeToString: func(b []byte) string { return string(b) },
		DecodeString:   func(s string) ([]byte, error) { return []byte(s), nil },
	},
	"16": {
		EncodeToString: hex.EncodeToString,
		DecodeString:   hex.DecodeString,
	},
	"32": {
		EncodeToString: base32.StdEncoding.EncodeToString,
		DecodeString:   base32.StdEncoding.DecodeString,
	},
	"32hex": {
		label:          "Base 32 \"Extended Hex Alphabet\"",
		EncodeToString: base32.HexEncoding.EncodeToString,
		DecodeString:   base32.HexEncoding.DecodeString,
	},
	"58": {
		EncodeToString: basen.Base58.EncodeToString,
		DecodeString:   basen.Base58.DecodeString,
	},
	"62": {
		EncodeToString: basen.Base62.EncodeToString,
		DecodeString:   basen.Base62.DecodeString,
	},
	"64": {
		EncodeToString: base64.StdEncoding.EncodeToString,
		DecodeString:   base64.StdEncoding.DecodeString,
	},
	"64url": {
		label:          "Base 64 alternate URL encoding",
		EncodeToString: base64.URLEncoding.EncodeToString,
		DecodeString:   base64.URLEncoding.DecodeString,
	},
}

func init() {
	var ok bool
	codecs["raw"], ok = codecs["none"]
	if !ok {
		panic("missing codec 'none'")
	}
	codecs["hex"], ok = codecs["16"]
	if !ok {
		panic("missing codec '16'")
	}
}

func die(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func showCodecs() {
	var lines []string
	log.Printf("supported codecs:")
	log.Printf("% 10s  %s", "codec", "description")
	log.Printf("----------- -------------------------------------------------------------")
	for k, v := range codecs {
		if v.label == "" {
			lines = append(lines, fmt.Sprintf("% 10s: base %s", k, k))
		} else {
			lines = append(lines, fmt.Sprintf("% 10s: %s", k, v.label))
		}
	}
	sort.Strings(lines)
	for _, line := range lines {
		log.Println(line)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("basen: ")
	flag.Parse()

	if *help {
		showCodecs()
		flag.PrintDefaults()
		die(nil)
	}

	var input io.Reader
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			die(err)
		}
		defer f.Close()
		input = f
	} else {
		input = os.Stdin
	}

	inCodec, ok := codecs[*decoder]
	if !ok {
		showCodecs()
		die(fmt.Errorf("unsupported codec: %q", *decoder))
	}
	outCodec, ok := codecs[*encoder]
	if !ok {
		showCodecs()
		die(fmt.Errorf("unsupported codec: %q", *encoder))
	}

	inputBuf, err := ioutil.ReadAll(input)
	if err != nil {
		die(err)
	}
	inputStr := string(inputBuf)
	if !inCodec.raw {
		inputStr = strings.TrimSpace(inputStr)
	}

	contents, err := inCodec.DecodeString(inputStr)
	if err != nil {
		die(err)
	}

	outputStr := outCodec.EncodeToString(contents)
	fmt.Print(outputStr)
	if !outCodec.raw {
		fmt.Print("\n")
	}
	die(nil)
}
