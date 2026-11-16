package rpc

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestGRPCUnaryInterceptorRecoversPanic(t *testing.T) {
	ipnet := &net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipnet})
	info := &grpc.UnaryServerInfo{FullMethod: "/Chain33/Test"}

	_, err := grpcUnaryInterceptor(ctx, nil, info, func(context.Context, interface{}) (interface{}, error) {
		panic("boom")
	})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))

	resp, err := grpcUnaryInterceptor(ctx, nil, info, func(context.Context, interface{}) (interface{}, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	require.Equal(t, "ok", resp)
}

func TestGRPCStreamInterceptorRecoversPanic(t *testing.T) {
	info := &grpc.StreamServerInfo{FullMethod: "/Chain33/SubEvent", IsServerStream: true}

	err := grpcStreamInterceptor(nil, nil, info, func(interface{}, grpc.ServerStream) error {
		panic("boom")
	})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))

	err = grpcStreamInterceptor(nil, nil, info, func(interface{}, grpc.ServerStream) error {
		return nil
	})
	require.NoError(t, err)
}
