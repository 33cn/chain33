package relayd

import (
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
	types.GrpcserviceClient
	closer io.Closer
}

func NewClient33(cfg *Chain33) *Client33 {
	address := cfg.Host + ":" + cfg.Endpoint
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client := types.NewGrpcserviceClient(conn)
	c := &Client33{
		config:            cfg,
		closer:            conn,
		GrpcserviceClient: client,
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

		case <-time.After(time.Second * 3):
			err := c.ping(ctx)
			if err != nil {
				log.Error("heartbeat", "heartbeat chain33 error: ", err.Error())
				c.AutoReconnect(ctx)
				reconnectAttempts--
			} else {
				reconnectAttempts = c.config.ReconnectAttempts
			}
			// TODO
			if reconnectAttempts < 0 {
				break out
			}
		}
	}
}

func (c *Client33) Start(ctx context.Context) error {
	go c.heartbeat(ctx)
	return nil
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

	if isSync.IsOk {
		c.isSyncing = isSync.IsOk
		log.Warn(fmt.Sprintf("node is syncing： %s", isSync.String()))
	}
	c.isSyncing = false
	return nil
}

func (c *Client33) AutoReconnect(ctx context.Context) {
	if c.isClosed && !c.config.DisableAutoReconnect {
		c.closer.Close()
		conn, err := grpc.Dial(c.config.Host+c.config.Endpoint, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}

		client := types.NewGrpcserviceClient(conn)
		c.closer = conn
		c.GrpcserviceClient = client
		c.isClosed = true
		c.Start(ctx)
	}
}

func (c *Client33) Close() error {
	return c.closer.Close()
}
