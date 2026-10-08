package grpcserver

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/LifeforDream/gometrics/internal/utils"
)

// ClientIPInterceptor забирает IP клиента и кладёт в контекст.
// Сначала проверяется metadata, потом адрес отправителя.
func ClientIPInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		var ipstr string
		if ok {
			ipmd := md.Get(utils.RealIPKey)
			for _, val := range ipmd {
				if val != "" {
					ipstr = strings.TrimSpace(val)
				}
			}
		}
		if ipstr == "" {
			p, ok := peer.FromContext(ctx)
			if ok {
				ipstr, _, _ = net.SplitHostPort(p.Addr.String())
			}
		}
		ctx = utils.WithClientIP(ctx, ipstr)
		return handler(ctx, req)
	}
}

// TrustedSubnetInterceptor проверяет вхождение IP клиента
// в доверенную подсеть network. В негатином случае возвращает ошибку доступа.
// Требует, чтобы ClientIPInterceptor выполнялся раньше.
func TrustedSubnetInterceptor(network *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if network == nil {
			return handler(ctx, req)
		}
		ip := net.ParseIP(utils.ClientIP(ctx))
		if ip == nil || !network.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "access from this IP is prohibited")
		}
		return handler(ctx, req)
	}
}
