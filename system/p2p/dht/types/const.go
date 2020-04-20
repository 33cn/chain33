// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package types

import (
	"errors"
	"time"
)

const (
	// P2P client version
	Version = "1.0.0"
	// DHTTypeName p2p插件名称，底层基于libp2p框架, dht结构化网络
	DHTTypeName = "dht"

	// DefaultP2PPort 默认端口
	DefaultP2PPort = 13803
)

var (
	ErrLength             = errors.New("length not equal")
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrNotFound           = errors.New("not found")
	ErrExpired            = errors.New("data expired")
	ErrInvalidParam       = errors.New("invalid parameter")
	ErrEmptyRoutingTable  = errors.New("empty routing table")
	ErrWrongSignature     = errors.New("wrong signature")

	//ExpiredTime     = time.Hour * 24
	//RefreshInterval = time.Hour * 4
	ExpiredTime     = time.Minute * 5
	RefreshInterval = time.Minute * 1
)
