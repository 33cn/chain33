package gocbcore

import (
	"sync"
)

type memdPipelineClient struct {
	parent     *memdPipeline
	clientDead bool
	client     *memdClient
	consumer   *memdOpConsumer
	lock       sync.Mutex
}

func newMemdPipelineClient(parent *memdPipeline) *memdPipelineClient {
	return &memdPipelineClient{
		parent: parent,
	}
}

func (pipecli *memdPipelineClient) ReassignTo(parent *memdPipeline) {
	pipecli.lock.Lock()
	pipecli.parent = parent
	consumer := pipecli.consumer
	pipecli.lock.Unlock()

	if consumer != nil {
		consumer.Close()
	}
}

func (pipecli *memdPipelineClient) ioLoop(client *memdClient) {
	pipecli.lock.Lock()
	if pipecli.parent == nil {
		logErrorf("Pipeline ioLoop started with no parent")
		pipecli.lock.Unlock()

		err := client.Close()
		if err != nil {
			logErrorf("Failed to shut down broken ioLoop client (%s)", err)
		}

		return
	}
	address := pipecli.parent.address
	pipecli.client = client
	pipecli.lock.Unlock()

	defer func() {
		pipecli.lock.Lock()
		pipecli.client = nil
		pipecli.lock.Unlock()
	}()

	killSig := make(chan struct{})

	go func() {
		logDebugf("Pipeline `%s/%p` client watcher starting...", address, pipecli)

		<-client.CloseNotify()

		logDebugf("Pipeline `%s/%p` client died", address, pipecli)

		pipecli.lock.Lock()
		pipecli.clientDead = true
		consumer := pipecli.consumer
		pipecli.lock.Unlock()

		logDebugf("Pipeline `%s/%p` closing consumer %p", address, pipecli, consumer)
		consumer.Close()

		killSig <- struct{}{}
	}()

	logDebugf("Pipeline `%s/%p` IO loop starting...", address, pipecli)

	var localConsumer *memdOpConsumer
	for {
		if localConsumer == nil {
			logDebugf("Pipeline `%s/%p` fetching new consumer", address, pipecli)

			pipecli.lock.Lock()

			if pipecli.parent == nil {
				logDebugf("Pipeline `%s/%p` found no parent", address, pipecli)

				// This pipelineClient has been shut down
				pipecli.lock.Unlock()
				break
			}

			if pipecli.clientDead {
				// The client has disconnected from the server
				pipecli.lock.Unlock()
				break
			}

			localConsumer = pipecli.parent.queue.Consumer()
			pipecli.consumer = localConsumer
			pipecli.lock.Unlock()
		}

		req := localConsumer.Pop()
		if req == nil {
			localConsumer = nil
			continue
		}

		err := client.SendRequest(req)
		if err != nil {
			logDebugf("Server write error: %v", err)

			// We need to alert the caller that there was a network error
			req.Callback(nil, req, ErrNetwork)

			// Stop looping
			break
		}
	}

	// Ensure the connection is fully closed
	err := client.Close()
	if err != nil {
		// We just log here, since this is a non-fatal error.
		logErrorf("Failed to shut down client socket (%s)", err)
	}

	// We must wait for the close wait goroutine to die as well before we can continue.
	<-killSig
}

func (pipecli *memdPipelineClient) Run() {
	for {
		logDebugf("Pipeline Client `%p` preparing for new client loop", pipecli)

		pipecli.lock.Lock()
		pipeline := pipecli.parent
		pipecli.clientDead = false
		pipecli.lock.Unlock()

		logDebugf("Pipeline Client `%p` is on parent: %p", pipecli, pipeline)

		if pipeline == nil {
			break
		}

		logDebugf("Pipeline Client `%p` retrieving new client connection", pipecli)
		client, err := pipeline.getClientFn()
		if err != nil {
			continue
		}

		// Runs until the connection has died (for whatever reason)
		logDebugf("Pipeline Client `%p` starting new client loop", pipecli)
		pipecli.ioLoop(client)
	}
}

func (pipecli *memdPipelineClient) Close() error {
	logDebugf("Pipeline Client `%p` received close request", pipecli)

	pipecli.lock.Lock()
	pipecli.parent = nil
	consumer := pipecli.consumer
	pipecli.lock.Unlock()

	if consumer != nil {
		consumer.Close()
	}

	return nil
}
