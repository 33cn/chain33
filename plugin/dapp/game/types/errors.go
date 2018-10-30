package types

import "errors"

var (
	ErrGameCreateAmount = errors.New("You fill in more than the maximum number of games.")
	ErrGameCancleAddr   = errors.New("You don't have permission to cancel someone else's game.")
	ErrGameCloseAddr    = errors.New("The game time has not yet expired,You don't have permission to call yet.")
	ErrGameTimeOut      = errors.New("The game has expired.,You don't have permission to call.")
	ErrGameMatchStatus  = errors.New("can't join the game, the game has matched or finished!")
	ErrGameMatch        = errors.New("can't join the game, You can't match the game you created!")
	ErrGameCancleStatus = errors.New("can't cancle the game, the game has matched!")
	ErrGameCloseStatus  = errors.New("can't close the game again, the game has  finished!")
)
