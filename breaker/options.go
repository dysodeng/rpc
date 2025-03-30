package breaker

import "time"

// Option 断路器配置选项
type Option func(*circuitBreaker)

// WithFailureThreshold 设置故障阈值
func WithFailureThreshold(threshold uint32) Option {
	return func(cb *circuitBreaker) {
		cb.failureThreshold = threshold
	}
}

// WithSuccessThreshold 设置成功阈值
func WithSuccessThreshold(threshold uint32) Option {
	return func(cb *circuitBreaker) {
		cb.successThreshold = threshold
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(cb *circuitBreaker) {
		cb.timeout = timeout
	}
}

// WithHalfOpenRetryInterval 设置半开状态重试间隔
func WithHalfOpenRetryInterval(interval time.Duration) Option {
	return func(cb *circuitBreaker) {
		cb.halfOpenRetryInterval = interval
	}
}

// WithIsFailureFunc 设置错误判断函数
func WithIsFailureFunc(f func(err error) bool) Option {
	return func(cb *circuitBreaker) {
		cb.isFailureFunc = f
	}
}
