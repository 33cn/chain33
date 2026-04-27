package tss

import (
	"context"
	"runtime"
	"sync"

	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// MessageHandler handle system message
type MessageHandler func(msg *MessageWrapper)

var (
	lock     sync.Mutex
	initOnce sync.Once
	handlers = make(map[string]MessageHandler)
	msgChan  = make(chan *MessageWrapper, 1024)
	log      = log15.New("module", "tss")
)

func init() {
	cryptocli.RegisterCryptoHandler(types.EventCryptoTssMsg, dispatchMessage)
	cryptocli.RegisterSubInitFunc("tss", initTSS)
}

// init tss service
func initTSS(ctx cryptocli.CryptoContext) error {
	cfg := ctx.API.GetConfig()
	if !cfg.GetModuleConfig().Crypto.EnableTSS {
		return nil
	}

	initOnce.Do(func() {

		for i := 0; i < 2*runtime.NumCPU(); i++ {
			go handleTssMsg(ctx.Ctx)
		}
	})

	return nil
}

// RegisterMsgHandler register handler
func RegisterMsgHandler(name string, handler MessageHandler) {
	lock.Lock()
	defer lock.Unlock()
	_, exist := handlers[name]
	if exist {
		panic("duplicate handler name=" + name)
	}
	handlers[name] = handler
}

func dispatchMessage(queueMsg *queue.Message) {
	msg, ok := queueMsg.Data.(*MessageWrapper)
	if !ok {
		log.Error("invalid message type")
		return
	}
	select {
	case msgChan <- msg:
	default:
		log.Error("msgChan is full", "discard msg", msg.Protocol, "session", msg.SessionID)
	}
}

func handleTssMsg(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		case wMsg := <-msgChan:
			handler, ok := handlers[wMsg.Protocol]
			if !ok {
				log.Error("handleTssMsg", "invalid protocol", wMsg.Protocol, "session", wMsg.SessionID)
				continue
			}
			handler(wMsg)
		}
	}
}
