// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ticket

var (
	// TicketPrefix ticket prefix
	TicketPrefix = []byte("mavl-ticket-")
)

const (
	// StatusNewTicket new ticket status
	StatusNewTicket = 1
	// StatusMinerTicket Miner ticket status
	StatusMinerTicket = 2
	// StatusCloseTicket Close ticket status
	StatusCloseTicket = 3
)
