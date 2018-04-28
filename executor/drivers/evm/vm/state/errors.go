package state

import "github.com/pkg/errors"

var (
	NoCoinsAccount = errors.New("no coins account in executor!")
)
