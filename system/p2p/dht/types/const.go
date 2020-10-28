// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//Package types dht public types
package types

import (
	"errors"
	"time"
)

const (
	//Version P2P client version
	Version = "1.0.0"
	// DHTTypeName p2p插件名称，底层基于libp2p框架, dht结构化网络
	DHTTypeName = "dht"

	// DefaultP2PPort 默认端口
	DefaultP2PPort = 13803
)

var (
	// ErrLength err length
	ErrLength = errors.New("length not equal")
	// ErrInvalidMessageType invalid message type err
	ErrInvalidMessageType = errors.New("invalid message type")
	// ErrNotFound not found err
	ErrNotFound = errors.New("not found")
	// ErrInvalidParam invalid param err
	ErrInvalidParam = errors.New("invalid parameter")
	// ErrWrongSignature wrong signature err
	ErrWrongSignature = errors.New("wrong signature")
	// ErrUnknown unknown err
	ErrUnknown = errors.New("unknown error")

	// ExpiredTime expired time
	ExpiredTime = time.Hour * 3
	// RefreshInterval refresh interval
	RefreshInterval = time.Hour
	// CheckHealthyInterval check healthy interval
	CheckHealthyInterval = time.Minute * 5
)
