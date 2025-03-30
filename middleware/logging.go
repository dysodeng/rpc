package middleware

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Logging(log *zap.Logger) UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		// 获取 grpc 状态码
		st, _ := status.FromError(err)
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Float64("latency_ms", float64(time.Since(startTime).Nanoseconds())/1e6),
			zap.String("status_code", st.Code().String()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("rpc request failed", fields...)
		} else {
			log.Info("rpc request completed", fields...)
		}

		return resp, err
	}
}
