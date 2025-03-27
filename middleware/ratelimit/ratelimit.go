package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是限流中间件的选项
type Option func(*options)

// Limiter 是限流器接口
type Limiter interface {
	// Allow 判断请求是否被允许
	Allow(ctx context.Context) (bool, error)
}

// WithLimiter 设置限流器
func WithLimiter(limiter Limiter) Option {
	return func(o *options) {
		o.limiter = limiter
	}
}

// options 是限流中间件的选项
type options struct {
	limiter Limiter
}

// RateLimit 返回一个限流中间件
func RateLimit(opts ...Option) middleware.Middleware {
	options := options{
		limiter: defaultLimiter{},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			allowed, err := options.limiter.Allow(ctx)
			if err != nil {
				return nil, err
			}
			if !allowed {
				return nil, errors.TooManyRequests("RATELIMIT", "too many requests")
			}
			return handler(ctx, req)
		}
	}
}

// defaultLimiter 是默认的限流器，允许所有请求通过
type defaultLimiter struct{}

// Allow 实现Limiter接口
func (l defaultLimiter) Allow(ctx context.Context) (bool, error) {
	return true, nil
}

// TokenBucket 是令牌桶限流器
type TokenBucket struct {
	rate       float64   // 每秒填充的令牌数
	capacity   float64   // 桶的容量
	tokens     float64   // 当前令牌数
	lastTokens time.Time // 上次添加令牌的时间
	mu         sync.Mutex
}

// NewTokenBucket 创建一个新的令牌桶限流器
func NewTokenBucket(rate, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastTokens: time.Now(),
	}
}

// Allow 实现Limiter接口
func (l *TokenBucket) Allow(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTokens).Seconds()
	l.lastTokens = now

	// 添加新令牌（不超过桶容量）
	l.tokens = min(l.capacity, l.tokens+elapsed*l.rate)

	if l.tokens < 1.0 {
		return false, nil
	}

	// 消耗一个令牌
	l.tokens -= 1.0
	return true, nil
}

// IPRateLimiter 是基于IP的限流器
type IPRateLimiter struct {
	limiters map[string]*TokenBucket
	mu       sync.Mutex
	rate     float64
	capacity float64
}

// NewIPRateLimiter 创建一个新的基于IP的限流器
func NewIPRateLimiter(rate, capacity float64) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

// Allow 实现Limiter接口
func (l *IPRateLimiter) Allow(ctx context.Context) (bool, error) {
	ip := getClientIP(ctx)
	if ip == "" {
		// 无法获取IP时，默认允许
		return true, nil
	}

	l.mu.Lock()
	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = NewTokenBucket(l.rate, l.capacity)
		l.limiters[ip] = limiter
	}
	l.mu.Unlock()

	return limiter.Allow(ctx)
}

// getClientIP 从上下文中获取客户端IP
func getClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip
	}
	return ""
}

// min 返回两个float64中的较小者
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
