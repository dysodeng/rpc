package retry

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Policy 重试策略
type Policy struct {
	// MaxAttempts 最大重试次数（包括首次请求）
	MaxAttempts uint
	// InitialBackoff 初始重试等待时间
	InitialBackoff time.Duration
	// MaxBackoff 最大重试等待时间
	MaxBackoff time.Duration
	// BackoffMultiplier 重试等待时间的增长倍数
	BackoffMultiplier float64
	// RetryableErrors 可重试的错误码列表
	RetryableErrors []codes.Code
}

// DefaultRetryPolicy 默认重试策略
var DefaultRetryPolicy = &Policy{
	MaxAttempts:       3,                      // 最多重试3次（包括首次请求）
	InitialBackoff:    100 * time.Millisecond, // 首次重试等待100ms
	MaxBackoff:        1 * time.Second,        // 最大重试等待时间1秒
	BackoffMultiplier: 2.0,                    // 每次重试等待时间是上次的2倍
	RetryableErrors: []codes.Code{
		codes.Unavailable,       // 服务不可用
		codes.DeadlineExceeded,  // 请求超时
		codes.ResourceExhausted, // 资源耗尽（如限流）
		codes.Internal,          // 内部错误
	},
}

// Interceptor 创建重试拦截器
func Interceptor(policy *Policy) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var lastErr error
		var backoff = policy.InitialBackoff

		for attempt := uint(0); attempt < policy.MaxAttempts; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err
			if st, ok := status.FromError(err); ok {
				retry := false
				for _, code := range policy.RetryableErrors {
					if st.Code() == code {
						retry = true
						break
					}
				}
				if !retry {
					return err
				}
			}

			// 计算下一次重试的等待时间
			if attempt == policy.MaxAttempts-1 {
				break
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff = time.Duration(float64(backoff) * policy.BackoffMultiplier)
				if backoff > policy.MaxBackoff {
					backoff = policy.MaxBackoff
				}
			}
		}

		return lastErr
	}
}
