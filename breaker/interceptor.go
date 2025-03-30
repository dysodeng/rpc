package breaker

import (
	"context"

	"google.golang.org/grpc"
)

// Interceptor 熔断器拦截器
func Interceptor(cb CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return cb.Execute(
			ctx,
			func() error {
				// 执行正常的 RPC 调用
				return invoker(ctx, method, req, reply, cc, opts...)
			},
			func(err error) error {
				// 服务降级处理
				// 这里可以返回默认值、缓存数据或者错误信息
				return err
			},
		)
	}
}
