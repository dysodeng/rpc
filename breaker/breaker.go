package breaker

import (
	"context"
	"sync"
	"time"
)

// State 断路器状态
type State int

const (
	StateClosed   State = iota // 关闭状态（允许请求）
	StateHalfOpen              // 半开状态（允许部分请求尝试）
	StateOpen                  // 开启状态（阻止请求）
)

// CircuitBreaker 断路器接口
type CircuitBreaker interface {
	// Execute 执行请求
	Execute(ctx context.Context, run func() error, fallback func(error) error) error
	// State 获取当前状态
	State() State
}

type circuitBreaker struct {
	mutex sync.RWMutex

	state                 State
	failureThreshold      uint32        // 故障阈值
	successThreshold      uint32        // 成功阈值
	failureCount          uint32        // 当前故障计数
	successCount          uint32        // 当前成功计数
	timeout               time.Duration // 超时时间
	lastStateChangeTime   time.Time     // 最后一次状态改变时间
	halfOpenRetryInterval time.Duration // 半开状态重试间隔
}

// NewCircuitBreaker 创建新的断路器
func NewCircuitBreaker(options ...Option) CircuitBreaker {
	cb := &circuitBreaker{
		state:                 StateClosed,
		failureThreshold:      5,                // 默认5次失败触发断路
		successThreshold:      3,                // 默认3次成功恢复服务
		timeout:               time.Second * 10, // 默认10秒超时
		halfOpenRetryInterval: time.Second * 5,  // 默认5秒重试间隔
	}

	for _, option := range options {
		option(cb)
	}

	return cb
}

func (cb *circuitBreaker) Execute(ctx context.Context, run func() error, fallback func(error) error) error {
	cb.mutex.RLock()
	state := cb.state
	cb.mutex.RUnlock()

	// 如果断路器打开，直接执行降级逻辑
	if state == StateOpen {
		if time.Since(cb.lastStateChangeTime) > cb.timeout {
			cb.mutex.Lock()
			cb.toHalfOpen()
			cb.mutex.Unlock()
		} else {
			return fallback(nil)
		}
	}

	// 执行请求
	err := run()

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		return cb.onFailure(err, fallback)
	}

	return cb.onSuccess()
}

func (cb *circuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

func (cb *circuitBreaker) onSuccess() error {
	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.toClosed()
		}
	case StateClosed:
		cb.failureCount = 0
	default:
	}
	return nil
}

func (cb *circuitBreaker) onFailure(err error, fallback func(error) error) error {
	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.toOpen()
		}
	case StateHalfOpen:
		cb.toOpen()
	default:
	}
	return fallback(err)
}

func (cb *circuitBreaker) toClosed() {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChangeTime = time.Now()
}

func (cb *circuitBreaker) toOpen() {
	cb.state = StateOpen
	cb.lastStateChangeTime = time.Now()
}

func (cb *circuitBreaker) toHalfOpen() {
	cb.state = StateHalfOpen
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChangeTime = time.Now()
}
