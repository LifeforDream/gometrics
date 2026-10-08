package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/LifeforDream/gometrics/internal/utils"
)

// capturingHandler возвращает grpc.UnaryHandler, который запоминает
// полученный ctx и отдаёт заданный ответ без дальнейшей обработки.
func capturingHandler(resp any, err error) (grpc.UnaryHandler, *context.Context) {
	var gotCtx context.Context
	handler := func(ctx context.Context, _ any) (any, error) {
		gotCtx = ctx
		return resp, err
	}
	return handler, &gotCtx
}

func TestClientIPInterceptor(t *testing.T) {
	tests := []struct {
		name   string
		ctx    func() context.Context
		wantIP string
	}{
		{
			name: "takes IP from x-real-ip metadata",
			ctx: func() context.Context {
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs(utils.RealIPKey, "10.0.0.5"))
			},
			wantIP: "10.0.0.5",
		},
		{
			name: "trims whitespace from metadata value",
			ctx: func() context.Context {
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs(utils.RealIPKey, "  10.0.0.5  "))
			},
			wantIP: "10.0.0.5",
		},
		{
			name: "falls back to peer address when metadata is empty",
			ctx: func() context.Context {
				ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(utils.RealIPKey, ""))
				return peer.NewContext(ctx, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 1234}})
			},
			wantIP: "1.2.3.4",
		},
		{
			name: "falls back to peer address when metadata is absent",
			ctx: func() context.Context {
				return peer.NewContext(context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 1234}})
			},
			wantIP: "1.2.3.4",
		},
		{
			name: "no metadata and no peer leaves IP empty",
			ctx: func() context.Context {
				return context.Background()
			},
			wantIP: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, gotCtx := capturingHandler("ok", nil)

			resp, err := ClientIPInterceptor()(tt.ctx(), "req", &grpc.UnaryServerInfo{}, handler)

			require.NoError(t, err)
			assert.Equal(t, "ok", resp)
			require.NotNil(t, *gotCtx)
			assert.Equal(t, tt.wantIP, utils.ClientIP(*gotCtx))
		})
	}
}

func TestTrustedSubnetInterceptor(t *testing.T) {
	_, trusted, err := net.ParseCIDR("10.0.0.0/24")
	require.NoError(t, err)

	tests := []struct {
		name       string
		network    *net.IPNet
		clientIP   string
		wantCalled bool
		wantCode   codes.Code
	}{
		{
			name:       "nil network passes through without checking IP",
			network:    nil,
			clientIP:   "",
			wantCalled: true,
		},
		{
			name:       "IP inside trusted subnet is allowed",
			network:    trusted,
			clientIP:   "10.0.0.42",
			wantCalled: true,
		},
		{
			name:       "IP outside trusted subnet is denied",
			network:    trusted,
			clientIP:   "192.168.0.1",
			wantCalled: false,
			wantCode:   codes.PermissionDenied,
		},
		{
			name:       "missing client IP in context is denied",
			network:    trusted,
			clientIP:   "",
			wantCalled: false,
			wantCode:   codes.PermissionDenied,
		},
		{
			name:       "unparseable client IP is denied",
			network:    trusted,
			clientIP:   "not-an-ip",
			wantCalled: false,
			wantCode:   codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := utils.WithClientIP(context.Background(), tt.clientIP)
			called := false
			handler := func(_ context.Context, _ any) (any, error) {
				called = true
				return "ok", nil
			}

			resp, err := TrustedSubnetInterceptor(tt.network)(ctx, "req", &grpc.UnaryServerInfo{}, handler)

			assert.Equal(t, tt.wantCalled, called)
			if tt.wantCalled {
				require.NoError(t, err)
				assert.Equal(t, "ok", resp)
				return
			}
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}
