package relayd

import (
	"errors"
	"gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
)

type Connections33 struct {
	config      *Config
	connections map[string]*connection
	quit        chan struct{}
}

func NewConnections33(config *Config) *Connections33 {
	connections := make(map[string]*connection, len(config.Chain33))
	for _, cfg := range config.Chain33 {
		conn, err := grpc.Dial(cfg.Host+cfg.Endpoint, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}

		client := types.NewGrpcserviceClient(conn)
		c := &connection{
			ID:                   cfg.id,
			User:                 cfg.User,
			Pass:                 cfg.Pass,
			DisableAutoReconnect: cfg.DisableAutoReconnect,
			closer:               conn,
			GrpcserviceClient:    client,
		}
		connections[c.ID] = c
	}

	m := Connections33{
		config:      config,
		connections: connections,
	}

	return &m
}

func (m *Connections33) Start() {

}

func (m *Connections33) Close() []error {
	errs := make([]error, 1)
	for _, c := range m.connections {
		if err := c.closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (m *Connections33) Ping(ctx context.Context) []error {
	errs := make([]error, 1)
	for _, c := range m.connections {
		if err := c.Ping(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

type connection struct {
	ID string
	types.GrpcserviceClient
	closer               io.Closer
	User                 string
	Pass                 string
	DisableAutoReconnect bool
	isSyncing            bool
	isClosed             bool
	lastHeader           int64
}

func (c *connection) Ping(ctx context.Context) error {
	lastHeader, err := c.GetLastHeader(ctx, nil)
	if err != nil {
		c.isClosed = false
		return err
	}
	c.isClosed = true
	c.lastHeader = lastHeader.Height

	isSync, err := c.IsSync(ctx, nil)
	if err != nil {
		return err
	}

	if isSync.IsOk {
		c.isSyncing = isSync.IsOk
		return errors.New(isSync.String())
	}

	c.isSyncing = false
	return nil
}
