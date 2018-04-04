package internal

import (
	"sync"
	"time"
)

type Message struct {
	topic   string
	content map[string]interface{}
	addedAt int64
}

type MessageQueue struct {
	MessageTTL    int64
	FlushInterval int64

	agent   *Agent
	started *Flag

	flushTimer *Timer

	queue              []Message
	queueLock          *sync.Mutex
	lastFlushTimestamp int64
	backoffSeconds     int64
}

func newMessageQueue(agent *Agent) *MessageQueue {
	mq := &MessageQueue{
		MessageTTL:    10 * 60,
		FlushInterval: 5,

		agent:   agent,
		started: &Flag{},

		flushTimer: nil,

		queue:              make([]Message, 0),
		queueLock:          &sync.Mutex{},
		lastFlushTimestamp: 0,
		backoffSeconds:     0,
	}

	return mq
}

func (mq *MessageQueue) start() {
	if !mq.started.SetIfUnset() {
		return
	}

	if mq.agent.AutoProfiling {
		mq.flushTimer = mq.agent.createTimer(0, time.Duration(mq.FlushInterval)*time.Second, func() {
			mq.flush()
		})
	}
}

func (mq *MessageQueue) stop() {
	if !mq.started.UnsetIfSet() {
		return
	}

	if mq.flushTimer != nil {
		mq.flushTimer.Stop()
	}
}

func (mq *MessageQueue) size() int {
	mq.queueLock.Lock()
	defer mq.queueLock.Unlock()

	return len(mq.queue)
}

func (mq *MessageQueue) expire() {
	mq.queueLock.Lock()
	defer mq.queueLock.Unlock()

	if len(mq.queue) == 0 {
		return
	}

	now := time.Now().Unix()

	if mq.queue[0].addedAt > now-mq.MessageTTL {
		return
	}

	for i := len(mq.queue) - 1; i >= 0; i-- {
		if mq.queue[i].addedAt < now-mq.MessageTTL {
			mq.queue = mq.queue[i+1:]
			break
		}
	}

	mq.agent.log("Expired old messages from the queue")
}

func (mq *MessageQueue) addMessage(topic string, message map[string]interface{}) {
	m := Message{
		topic:   topic,
		content: message,
		addedAt: time.Now().Unix(),
	}

	// add new message
	mq.queueLock.Lock()
	mq.queue = append(mq.queue, m)
	mq.queueLock.Unlock()

	mq.agent.log("Added message to the queue for topic: %v", topic)
	mq.agent.log("%v", message)

	mq.expire()
}

func (mq *MessageQueue) flush() {
	now := time.Now().Unix()
	if !mq.agent.AutoProfiling && mq.lastFlushTimestamp > now-mq.FlushInterval {
		return
	}

	if mq.size() == 0 {
		return
	}

	// flush only if backoff time is elapsed
	if mq.lastFlushTimestamp+mq.backoffSeconds > now {
		return
	}

	mq.expire()

	mq.queueLock.Lock()
	outgoing := mq.queue
	mq.queue = make([]Message, 0)
	mq.queueLock.Unlock()

	messages := make([]interface{}, 0)
	for _, m := range outgoing {
		message := map[string]interface{}{
			"topic":   m.topic,
			"content": m.content,
		}

		messages = append(messages, message)
	}

	payload := map[string]interface{}{
		"messages": messages,
	}

	mq.lastFlushTimestamp = now

	if _, err := mq.agent.apiRequest.post("upload", payload); err == nil {
		// reset backoff
		mq.backoffSeconds = 0
	} else {
		// prepend outgoing messages back to the queue
		mq.queueLock.Lock()
		mq.queue = append(outgoing, mq.queue...)
		mq.queueLock.Unlock()

		// increase backoff up to 1 minute
		mq.agent.log("Error uploading messages to dashboard, backing off next upload")
		if mq.backoffSeconds == 0 {
			mq.backoffSeconds = 10
		} else if mq.backoffSeconds*2 < 60 {
			mq.backoffSeconds *= 2
		}

		mq.agent.error(err)
	}
}

func (mq *MessageQueue) read() []Message {
	mq.queueLock.Lock()
	defer mq.queueLock.Unlock()

	messages := mq.queue
	mq.queue = make([]Message, 0)

	return messages
}
