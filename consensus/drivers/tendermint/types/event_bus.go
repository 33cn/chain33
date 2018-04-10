package types

import (
	"context"
	"github.com/tendermint/tmlibs/log"
)

const defaultCapacity = 1000

type EventBusSubscriber interface {
	Subscribe(ctx context.Context, subscriber string, query Query, out chan<- interface{}) error
	Unsubscribe(ctx context.Context, subscriber string, query Query) error
	UnsubscribeAll(ctx context.Context, subscriber string) error
}

// EventBus is a common bus for all events going through the system. All calls
// are proxied to underlying pubsub server. All events must be published using
// EventBus to ensure correct data types.
type EventBus struct {
	pubsub *Server
}

// NewEventBus returns a new event bus.
func NewEventBus() *EventBus {
	return NewEventBusWithBufferCapacity(defaultCapacity)
}

// NewEventBusWithBufferCapacity returns a new event bus with the given buffer capacity.
func NewEventBusWithBufferCapacity(cap int) *EventBus {
	// capacity could be exposed later if needed
	pubsub := NewServer(BufferCapacity(cap))
	b := &EventBus{pubsub: pubsub}
	return b
}

/*
func (b *EventBus) SetLogger(l log.Logger) {
	b.BaseService.SetLogger(l)
	b.pubsub.SetLogger(l.With("module", "pubsub"))
}
*/

func (b *EventBus) Start() error {
	return b.pubsub.Start()
}

func (b *EventBus) Stop() {
	b.pubsub.Stop()
}

func (b *EventBus) Subscribe(ctx context.Context, subscriber string, query Query, out chan<- interface{}) error {
	return b.pubsub.Subscribe(ctx, subscriber, query, out)
}

func (b *EventBus) Unsubscribe(ctx context.Context, subscriber string, query Query) error {
	return b.pubsub.Unsubscribe(ctx, subscriber, query)
}

func (b *EventBus) UnsubscribeAll(ctx context.Context, subscriber string) error {
	return b.pubsub.UnsubscribeAll(ctx, subscriber)
}

func (b *EventBus) Publish(eventType string, eventData TMEventData) error {
	// no explicit deadline for publishing events
	ctx := context.Background()
	b.pubsub.PublishWithTags(ctx, eventData, map[string]interface{}{EventTypeKey: eventType})
	return nil
}

//--- block, tx, and vote events

func (b *EventBus) PublishEventNewBlock(event EventDataNewBlock) error {
	return b.Publish(EventNewBlock, TMEventData{event})
}

func (b *EventBus) PublishEventNewBlockHeader(event EventDataNewBlockHeader) error {
	return b.Publish(EventNewBlockHeader, TMEventData{event})
}

func (b *EventBus) PublishEventVote(event EventDataVote) error {
	return b.Publish(EventVote, TMEventData{event})
}

// PublishEventTx publishes tx event with tags from Result. Note it will add
// predefined tags (EventTypeKey, TxHashKey). Existing tags with the same names
// will be overwritten.
func (b *EventBus) PublishEventTx(event EventDataTx) error {
	// no explicit deadline for publishing events
	ctx := context.Background()

	tags := make(map[string]interface{})

	//20180226 hg do it later
	/*
	// validate and fill tags from tx result
	for _, tag := range event.Result.Tags {
		// basic validation
		if tag.Key == "" {
			b.Logger.Info("Got tag with an empty key (skipping)", "tag", tag, "tx", event.Tx)
			continue
		}

		switch tag.ValueType {
		case abci.KVPair_STRING:
			tags[tag.Key] = tag.ValueString
		case abci.KVPair_INT:
			tags[tag.Key] = tag.ValueInt
		}
	}

	// add predefined tags
	logIfTagExists(EventTypeKey, tags, b.Logger)
	tags[EventTypeKey] = EventTx

	logIfTagExists(TxHashKey, tags, b.Logger)
	tags[TxHashKey] = fmt.Sprintf("%X", event.Tx.Hash())

	logIfTagExists(TxHeightKey, tags, b.Logger)
	tags[TxHeightKey] = event.Height
	*/

	b.pubsub.PublishWithTags(ctx, TMEventData{event}, tags)
	return nil
}

func (b *EventBus) PublishEventProposalHeartbeat(event EventDataProposalHeartbeat) error {
	return b.Publish(EventProposalHeartbeat, TMEventData{event})
}

//--- EventDataRoundState events

func (b *EventBus) PublishEventNewRoundStep(event EventDataRoundState) error {
	return b.Publish(EventNewRoundStep, TMEventData{event})
}

func (b *EventBus) PublishEventTimeoutPropose(event EventDataRoundState) error {
	return b.Publish(EventTimeoutPropose, TMEventData{event})
}

func (b *EventBus) PublishEventTimeoutWait(event EventDataRoundState) error {
	return b.Publish(EventTimeoutWait, TMEventData{event})
}

func (b *EventBus) PublishEventNewRound(event EventDataRoundState) error {
	return b.Publish(EventNewRound, TMEventData{event})
}

func (b *EventBus) PublishEventCompleteProposal(event EventDataRoundState) error {
	return b.Publish(EventCompleteProposal, TMEventData{event})
}

func (b *EventBus) PublishEventPolka(event EventDataRoundState) error {
	return b.Publish(EventPolka, TMEventData{event})
}

func (b *EventBus) PublishEventUnlock(event EventDataRoundState) error {
	return b.Publish(EventUnlock, TMEventData{event})
}

func (b *EventBus) PublishEventRelock(event EventDataRoundState) error {
	return b.Publish(EventRelock, TMEventData{event})
}

func (b *EventBus) PublishEventLock(event EventDataRoundState) error {
	return b.Publish(EventLock, TMEventData{event})
}

func logIfTagExists(tag string, tags map[string]interface{}, logger log.Logger) {
	if value, ok := tags[tag]; ok {
		logger.Error("Found predefined tag (value will be overwritten)", "tag", tag, "value", value)
	}
}
