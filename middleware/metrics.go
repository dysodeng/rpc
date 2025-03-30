package middleware

import (
	"context"
	"time"

	"github.com/dysodeng/rpc/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Metrics() UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		st, _ := status.FromError(err)
		metrics.RecordRequest(
			ctx,
			info.FullMethod,
			st.Code().String(),
			time.Since(startTime).Seconds(),
		)

		return resp, err
	}
}
