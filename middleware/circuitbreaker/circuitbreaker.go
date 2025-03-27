package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
)

// State 定义断路器状态
type State int

const (
	// StateClosed 表示断路器关闭（允许请求通过）
	StateClosed State = iota
	// StateOpen 表示断路器打开（阻止请求通过）
	StateOpen
	// StateHalfOpen 表示断路器半开（允许有限请求通过以测试服务）
	StateHalfOpen
)

// String 返回状态字符串表示
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Option 是断路器中间件的选项
type Option func(*options)

// WithTimeout 设置断路器打开状态的超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithThreshold 设置触发断路器的错误阈值
func WithThreshold(threshold int) Option {
	return func(o *options) {
		o.threshold = threshold
	}
}

// WithOnStateChange 设置状态变化回调函数
func WithOnStateChange(fn func(name string, from, to State)) Option {
	return func(o *options) {
		o.onStateChange = fn
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithMaxRequests 设置半开状态下允许的最大请求数
func WithMaxRequests(n int) Option {
	return func(o *options) {
		o.maxRequests = n
	}
}

// WithName 设置断路器名称
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// options 是断路器中间件的选项
type options struct {
	name          string
	timeout       time.Duration
	threshold     int
	maxRequests   int
	onStateChange func(name string, from, to State)
	logger        log.Logger
}

// Breaker 断路器中间件
type Breaker struct {
	name          string
	timeout       time.Duration
	threshold     int
	maxRequests   int
	onStateChange func(name string, from, to State)
	logger        log.Logger

	mutex       sync.RWMutex
	state       State
	failures    int
	lastFailure time.Time
	successes   int
}

// NewBreaker 创建一个新的断路器
func NewBreaker(opts ...Option) *Breaker {
	options := options{
		name:        "default",
		timeout:     time.Second * 30,
		threshold:   5,
		maxRequests: 3,
		onStateChange: func(name string, from, to State) {
			// 默认不处理
		},
		logger: log.DefaultLogger,
	}
	for _, o := range opts {
		o(&options)
	}

	return &Breaker{
		name:          options.name,
		timeout:       options.timeout,
		threshold:     options.threshold,
		maxRequests:   options.maxRequests,
		onStateChange: options.onStateChange,
		logger:        options.logger,
		state:         StateClosed,
	}
}

// Execute 执行断路器保护的操作
func (cb *Breaker) Execute(ctx context.Context, req interface{}, handler middleware.Handler) (interface{}, error) {
	// 检查断路器状态
	if !cb.AllowRequest() {
		cb.logger.Error("Circuit breaker open, request rejected",
			log.String("name", cb.name),
			log.String("state", cb.state.String()),
		)
		return nil, errors.ServiceUnavailable("CIRCUIT_OPEN", "service unavailable due to circuit breaker")
	}

	resp, err := handler(ctx, req)

	// 更新断路器状态
	if err != nil {
		cb.RegisterFailure()
	} else {
		cb.RegisterSuccess()
	}

	return resp, err
}

// Middleware 返回断路器中间件
func (cb *Breaker) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			return cb.Execute(ctx, req, handler)
		}
	}
}

// AllowRequest 判断是否允许请求通过断路器
func (cb *Breaker) AllowRequest() bool {
	cb.mutex.RLock()
	state := cb.state
	cb.mutex.RUnlock()

	// 断路器关闭，允许所有请求
	if state == StateClosed {
		return true
	}

	// 断路器打开，检查是否超时
	if state == StateOpen {
		openDuration := time.Since(cb.lastFailure)
		if openDuration >= cb.timeout {
			// 超时，转为半开状态
			cb.mutex.Lock()
			if cb.state == StateOpen {
				cb.changeState(StateHalfOpen)
				cb.successes = 0
			}
			cb.mutex.Unlock()
			return true
		}
		return false
	}

	// 断路器半开，检查请求数量
	if state == StateHalfOpen {
		cb.mutex.RLock()
		defer cb.mutex.RUnlock()
		return cb.successes < cb.maxRequests
	}

	return true
}

// RegisterSuccess 注册成功请求
func (cb *Breaker) RegisterSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 只在半开状态下计数成功请求
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.maxRequests {
			cb.changeState(StateClosed)
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// RegisterFailure 注册失败请求
func (cb *Breaker) RegisterFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// 更新失败计数和时间
	cb.failures++
	cb.lastFailure = time.Now()

	// 在关闭状态下，检查是否超过阈值
	if cb.state == StateClosed && cb.failures >= cb.threshold {
		cb.changeState(StateOpen)
	}

	// 在半开状态下，任何失败都会重新打开断路器
	if cb.state == StateHalfOpen {
		cb.changeState(StateOpen)
		cb.successes = 0
	}
}

// State 获取断路器当前状态
func (cb *Breaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// changeState 更改断路器状态并触发回调
func (cb *Breaker) changeState(to State) {
	from := cb.state
	cb.state = to

	// 记录状态变化
	cb.logger.Info("Circuit breaker state changed",
		log.String("name", cb.name),
		log.String("from", from.String()),
		log.String("to", to.String()),
	)

	// 触发回调
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, from, to)
	}
}

// CircuitBreaker 返回一个断路器中间件
func CircuitBreaker(opts ...Option) middleware.Middleware {
	cb := NewBreaker(opts...)
	return cb.Middleware()
}
