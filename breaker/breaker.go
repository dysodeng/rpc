package breaker

import (
	"context"
	"errors"
	"sync"
	"time"

	rpcError "github.com/dysodeng/rpc/errors"
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
	failureThreshold      uint32
	successThreshold      uint32
	failureCount          uint32
	successCount          uint32
	timeout               time.Duration
	lastStateChangeTime   time.Time
	halfOpenRetryInterval time.Duration
	isFailureFunc         func(err error) bool // 添加错误判断函数
}

func NewCircuitBreaker(options ...Option) CircuitBreaker {
	cb := &circuitBreaker{
		state:                 StateClosed,
		failureThreshold:      5,                // 默认5次失败触发断路
		successThreshold:      3,                // 默认3次成功恢复服务
		timeout:               time.Second * 10, // 默认10秒超时
		halfOpenRetryInterval: time.Second * 5,  // 默认5秒重试间隔
		isFailureFunc:         DefaultIsFailure, // 设置默认错误判断函数
	}

	for _, option := range options {
		option(cb)
	}

	return cb
}

// DefaultIsFailure 默认错误判断函数
func DefaultIsFailure(err error) bool {
	if err == nil {
		return false
	}

	// 转换为 rpc 错误
	rpcErr := rpcError.ToError(err)
	var e *rpcError.Error
	if errors.As(rpcErr, &e) {
		// 业务错误码范围判断
		if e.Code >= 11000 && e.Code < 20000 {
			return false // 业务错误不计入失败
		}
	}
	return true // 其他错误计入失败
}

func (cb *circuitBreaker) Execute(ctx context.Context, run func() error, fallback func(error) error) error {
	cb.mutex.RLock()
	state := cb.state
	cb.mutex.RUnlock()

	// 如果断路器打开，先判断是否需要进入半开状态
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

	// 如果是业务错误，直接返回，不影响断路器状态
	if err != nil && !cb.isFailureFunc(err) {
		return err
	}

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
