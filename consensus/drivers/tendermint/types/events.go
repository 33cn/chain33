package types

import (
	"github.com/tendermint/go-wire/data"
)

// Reserved event types
const (
	EventBond              = "Bond"
	EventCompleteProposal  = "CompleteProposal"
	EventDupeout           = "Dupeout"
	EventFork              = "Fork"
	EventLock              = "Lock"
	EventNewBlock          = "NewBlock"
	EventNewBlockHeader    = "NewBlockHeader"
	EventNewRound          = "NewRound"
	EventNewRoundStep      = "NewRoundStep"
	EventPolka             = "Polka"
	EventRebond            = "Rebond"
	EventRelock            = "Relock"
	EventTimeoutPropose    = "TimeoutPropose"
	EventTimeoutWait       = "TimeoutWait"
	EventTx                = "Tx"
	EventUnbond            = "Unbond"
	EventUnlock            = "Unlock"
	EventVote              = "Vote"
	EventProposalHeartbeat = "ProposalHeartbeat"
)

///////////////////////////////////////////////////////////////////////////////
// ENCODING / DECODING
///////////////////////////////////////////////////////////////////////////////

var (
	EventDataNameNewBlock          = "new_block"
	EventDataNameNewBlockHeader    = "new_block_header"
	EventDataNameTx                = "tx"
	EventDataNameRoundState        = "round_state"
	EventDataNameVote              = "vote"
	EventDataNameProposalHeartbeat = "proposal_heartbeat"
)

// implements events.EventData
type TMEventDataInner interface {
	// empty interface
}

type TMEventData struct {
	TMEventDataInner `json:"unwrap"`
}

func (tmr TMEventData) MarshalJSON() ([]byte, error) {
	return tmEventDataMapper.ToJSON(tmr.TMEventDataInner)
}

func (tmr *TMEventData) UnmarshalJSON(data []byte) (err error) {
	parsed, err := tmEventDataMapper.FromJSON(data)
	if err == nil && parsed != nil {
		tmr.TMEventDataInner = parsed.(TMEventDataInner)
	}
	return
}

func (tmr TMEventData) Unwrap() TMEventDataInner {
	tmrI := tmr.TMEventDataInner
	for wrap, ok := tmrI.(TMEventData); ok; wrap, ok = tmrI.(TMEventData) {
		tmrI = wrap.TMEventDataInner
	}
	return tmrI
}

func (tmr TMEventData) Empty() bool {
	return tmr.TMEventDataInner == nil
}

const (
	EventDataTypeNewBlock          = byte(0x01)
	EventDataTypeFork              = byte(0x02)
	EventDataTypeTx                = byte(0x03)
	EventDataTypeNewBlockHeader    = byte(0x04)
	EventDataTypeRoundState        = byte(0x11)
	EventDataTypeVote              = byte(0x12)
	EventDataTypeProposalHeartbeat = byte(0x20)
)

var tmEventDataMapper = data.NewMapper(TMEventData{}).
	RegisterImplementation(EventDataNewBlock{}, EventDataNameNewBlock, EventDataTypeNewBlock).
	RegisterImplementation(EventDataNewBlockHeader{}, EventDataNameNewBlockHeader, EventDataTypeNewBlockHeader).
	RegisterImplementation(EventDataTx{}, EventDataNameTx, EventDataTypeTx).
	RegisterImplementation(EventDataRoundState{}, EventDataNameRoundState, EventDataTypeRoundState).
	RegisterImplementation(EventDataVote{}, EventDataNameVote, EventDataTypeVote).
	RegisterImplementation(EventDataProposalHeartbeat{}, EventDataNameProposalHeartbeat, EventDataTypeProposalHeartbeat)

// Most event messages are basic types (a block, a transaction)
// but some (an input to a call tx or a receive) are more exotic

type EventDataNewBlock struct {
	Block *Block `json:"block"`
}

// light weight event for benchmarking
type EventDataNewBlockHeader struct {
	Header *Header `json:"header"`
}

// All txs fire EventDataTx
type EventDataTx struct {
	TxResult
}

type EventDataProposalHeartbeat struct {
	Heartbeat *Heartbeat
}

// NOTE: This goes into the replay WAL
type EventDataRoundState struct {
	Height int64  `json:"height"`
	Round  int    `json:"round"`
	Step   string `json:"step"`

	// private, not exposed to websockets
	RoundState interface{} `json:"-"`
}

type EventDataVote struct {
	Vote *Vote
}

///////////////////////////////////////////////////////////////////////////////
// PUBSUB
///////////////////////////////////////////////////////////////////////////////

const (
	// EventTypeKey is a reserved key, used to specify event type in tags.
	EventTypeKey = "tm.event"
	// TxHashKey is a reserved key, used to specify transaction's hash.
	// see EventBus#PublishEventTx
	TxHashKey = "tx.hash"
	// TxHeightKey is a reserved key, used to specify transaction block's height.
	// see EventBus#PublishEventTx
	TxHeightKey = "tx.height"
)

// BlockEventPublisher publishes all block related events
type BlockEventPublisher interface {
	PublishEventNewBlock(block EventDataNewBlock) error
	PublishEventNewBlockHeader(header EventDataNewBlockHeader) error
	PublishEventTx(EventDataTx) error
}

type TxEventPublisher interface {
	PublishEventTx(EventDataTx) error
}
