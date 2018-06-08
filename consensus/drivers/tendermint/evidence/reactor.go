package evidence

import (
	"fmt"
	"time"

	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/p2p"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"encoding/json"
)

const (
	EvidenceChannel = byte(0x38)

	maxEvidenceMessageSize     = 1048576 // 1MB TODO make it configurable
	broadcastEvidenceIntervalS = 60      // broadcast uncommitted evidence this often
)

// EvidenceReactor handles evpool evidence broadcasting amongst peers.
type EvidenceReactor struct {
	p2p.BaseReactor
	evpool   *EvidencePool
	eventBus *types.EventBus
	Logger   log.Logger
	Quit     chan struct{}
}

// NewEvidenceReactor returns a new EvidenceReactor with the given config and evpool.
func NewEvidenceReactor(evpool *EvidencePool) *EvidenceReactor {
	evR := &EvidenceReactor{
		evpool: evpool,
	}
	evR.BaseReactor = *p2p.NewBaseReactor("EvidenceReactor", evR)
	return evR
}

// SetLogger sets the Logger on the reactor and the underlying Evidence.
func (evR *EvidenceReactor) SetLogger(l log.Logger) {
	evR.Logger = l
	evR.evpool.SetLogger(l)
}

// OnStart implements cmn.Service
func (evR *EvidenceReactor) OnStart() error {
	if err := evR.BaseReactor.OnStart(); err != nil {
		return err
	}

	go evR.broadcastRoutine()
	return nil
}

// GetChannels implements Reactor.
// It returns the list of channels for this reactor.
func (evR *EvidenceReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:       EvidenceChannel,
			Priority: 5,
		},
	}
}

// AddPeer implements Reactor.
func (evR *EvidenceReactor) AddPeer(peer p2p.Peer) {
	// send the peer our high-priority evidence.
	// the rest will be sent by the broadcastRoutine
	evidences := evR.evpool.PriorityEvidence()
	msg := &types.EvidenceListMessage{evidences}
	success := peer.Send(EvidenceChannel, msg)
	if !success {
		// TODO: remove peer ?
	}
}

// RemovePeer implements Reactor.
func (evR *EvidenceReactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	// nothing to do
}

// Receive implements Reactor.
// It adds any received evidence to the evpool.
func (evR *EvidenceReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	envelope := types.MsgEnvelope{}
	err := json.Unmarshal(msgBytes, &envelope)
	if err != nil {
		evR.Logger.Error("Error decoding message", "err", err)
		return
	}
	evR.Logger.Debug("Receive", "src", src, "chId", chID)

	if v, ok := types.MsgMap[envelope.Kind]; ok {
		msg := v.(types.ReactorMsg).Copy()
		err = json.Unmarshal(*envelope.Data, &msg)
		if err != nil {
			evR.Logger.Error("EvidenceReactor Receive Unmarshal data failed:%v\n", err)
			return
		}
		switch msg := msg.(type) {
		case *types.EvidenceListMessage:
			for _, ev := range msg.Evidence {
				err := evR.evpool.AddEvidence(ev)
				if err != nil {
					evR.Logger.Error("Evidence is not valid", "evidence", msg.Evidence, "err", err)
					// TODO: punish peer
				}
			}
		default:
			evR.Logger.Error(fmt.Sprintf("Unknown message type %v", msg.TypeName()))
		}
	} else {
		evR.Logger.Error("not find ReactorMsg kind", "kind", envelope.Kind)
	}
}

// SetEventSwitch implements events.Eventable.
func (evR *EvidenceReactor) SetEventBus(b *types.EventBus) {
	evR.eventBus = b
}

// Broadcast new evidence to all peers.
// Broadcasts must be non-blocking so routine is always available to read off EvidenceChan.
func (evR *EvidenceReactor) broadcastRoutine() {
	ticker := time.NewTicker(time.Second * broadcastEvidenceIntervalS)
	for {
		select {
		case evidence := <-evR.evpool.EvidenceChan():
			// broadcast some new evidence
			msg := &types.EvidenceListMessage{[]types.Evidence{evidence}}
			evR.Switch.Broadcast(EvidenceChannel, msg)

			// TODO: Broadcast runs asynchronously, so this should wait on the successChan
			// in another routine before marking to be proper.
			evR.evpool.evidenceStore.MarkEvidenceAsBroadcasted(evidence)
		case <-ticker.C:
			// broadcast all pending evidence
			msg := &types.EvidenceListMessage{evR.evpool.PendingEvidence()}
			evR.Switch.Broadcast(EvidenceChannel, msg)
		case <-evR.Quit:
			return
		}
	}
}
