package relayd

import (
	"errors"
	"fmt"
	"io"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client33 struct {
	config     *Chain33
	isSyncing  bool
	isClosed   bool
	lastHeight int64
	types.Chain33Client
	closer io.Closer
}

func NewClient33(cfg *Chain33) *Client33 {
	conn, err := grpc.Dial(cfg.Host, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client := types.NewChain33Client(conn)
	c := &Client33{
		config:        cfg,
		closer:        conn,
		Chain33Client: client,
	}
	return c
}

func (c *Client33) heartbeat(ctx context.Context) {
	reconnectAttempts := c.config.ReconnectAttempts
out:
	for {
		log.Info("chain33 heartbeat.......")
		select {
		case <-ctx.Done():
			break out

		case <-time.After(time.Second * 60):
			err := c.ping(ctx)
			if err != nil {
				log.Error("heartbeat", "heartbeat chain33 error: ", err.Error(), "reconnectAttempts: ", reconnectAttempts)
				c.AutoReconnect(ctx)
				reconnectAttempts--
			} else {
				reconnectAttempts = c.config.ReconnectAttempts
			}
			// TODO
			if reconnectAttempts <= 0 {
				panic(fmt.Errorf("reconnectAttempts <= %d", reconnectAttempts))
			}
		}
	}
}

func (c *Client33) Start(ctx context.Context) {
	go c.heartbeat(ctx)
}

func (c *Client33) ping(ctx context.Context) error {
	lastHeader, err := c.GetLastHeader(ctx, &types.ReqNil{})
	if err != nil {
		c.isClosed = false
		return err
	}

	c.isClosed = true
	c.lastHeight = lastHeader.Height
	log.Info("ping", "lastHeight:", c.lastHeight)
	isSync, err := c.IsSync(ctx, &types.ReqNil{})
	if err != nil {
		return err
	}

	if !isSync.IsOk {
		c.isSyncing = !isSync.IsOk
		log.Warn(fmt.Sprintf("node is syncing： %s", isSync.String()))
	}
	c.isSyncing = false
	return nil
}

func (c *Client33) AutoReconnect(ctx context.Context) {
	if c.isClosed && !c.config.DisableAutoReconnect {
		c.closer.Close()
		conn, err := grpc.Dial(c.config.Host, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}

		client := types.NewChain33Client(conn)
		c.closer = conn
		c.Chain33Client = client
		c.isClosed = true
		c.Start(ctx)
	}
}

func (c *Client33) SendTransaction(ctx context.Context, in *types.Transaction) (*types.Reply, error) {
	if c.isSyncing {
		return nil, errors.New("node is syncing")
	}
	return c.Chain33Client.SendTransaction(ctx, in)
}

func (c *Client33) Close() error {
	return c.closer.Close()
}
