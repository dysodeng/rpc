package limiter

import (
	"context"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	limiter *rate.Limiter
}

// NewTokenBucketLimiter 创建新的令牌桶限流器
// @params tokenRate 令牌产生速率（每秒产生的令牌数）
// @params bucketSize 令牌桶容量（最大可存储的令牌数）
func NewTokenBucketLimiter(tokenRate float64, bucketSize int) *TokenBucketLimiter {
	// 确保最小速率不小于每秒1个请求
	if tokenRate < 1 {
		tokenRate = 1
	}
	return &TokenBucketLimiter{
		limiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(tokenRate)), bucketSize),
	}
}

func (l *TokenBucketLimiter) Allow() bool {
	return l.limiter.Allow()
}

func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

// RateLimitInterceptor 创建限流拦截器
func RateLimitInterceptor(limiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := limiter.Wait(ctx); err != nil {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
