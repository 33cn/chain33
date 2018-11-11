// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows,!plan9

package log15

import (
	"io"
	"log/syslog"
	"strings"
)

// SyslogHandler opens a connection to the system syslog daemon by calling
// syslog.New and writes all records to it.
func SyslogHandler(priority syslog.Priority, tag string, fmtr Format) (Handler, error) {
	wr, err := syslog.New(priority, tag)
	return sharedSyslog(fmtr, wr, err)
}

// SyslogNetHandler opens a connection to a log daemon over the network and writes
// all log records to it.
func SyslogNetHandler(net, addr string, priority syslog.Priority, tag string, fmtr Format) (Handler, error) {
	wr, err := syslog.Dial(net, addr, priority, tag)
	return sharedSyslog(fmtr, wr, err)
}

func sharedSyslog(fmtr Format, sysWr io.WriteCloser, err error) (Handler, error) {
	if err != nil {
		return nil, err
	}
	wr := sysWr.(*syslog.Writer)
	h := FuncHandler(int(LvlDebug), func(r *Record) error {
		var syslogFn = wr.Info
		switch r.Lvl {
		case LvlCrit:
			syslogFn = wr.Crit
		case LvlError:
			syslogFn = wr.Err
		case LvlWarn:
			syslogFn = wr.Warning
		case LvlInfo:
			syslogFn = wr.Info
		case LvlDebug:
			syslogFn = wr.Debug
		}
		s := strings.TrimSpace(string(fmtr.Format(r)))
		return syslogFn(s)
	})
	return LazyHandler(&closingHandler{sysWr, h}), nil
}

func (m muster) SyslogHandler(priority syslog.Priority, tag string, fmtr Format) Handler {
	return must(SyslogHandler(priority, tag, fmtr))
}

func (m muster) SyslogNetHandler(net, addr string, priority syslog.Priority, tag string, fmtr Format) Handler {
	return must(SyslogNetHandler(net, addr, priority, tag, fmtr))
}
