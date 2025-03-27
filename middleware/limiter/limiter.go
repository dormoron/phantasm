package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是限流中间件的选项
type Option func(*options)

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithKeyFunc 设置键生成函数
func WithKeyFunc(keyFunc KeyFunc) Option {
	return func(o *options) {
		o.keyFunc = keyFunc
	}
}

// WithLimiter 设置限流器实现
func WithLimiter(limiter Limiter) Option {
	return func(o *options) {
		o.limiter = limiter
	}
}

// WithFailureHandler 设置失败处理函数
func WithFailureHandler(handler func(ctx context.Context, req interface{}) (interface{}, error)) Option {
	return func(o *options) {
		o.failureHandler = handler
	}
}

// options 是限流中间件的选项
type options struct {
	logger         log.Logger
	keyFunc        KeyFunc
	limiter        Limiter
	failureHandler func(ctx context.Context, req interface{}) (interface{}, error)
}

// KeyFunc 定义键生成函数类型
type KeyFunc func(ctx context.Context) string

// defaultKeyFunc 默认的键生成函数
func defaultKeyFunc(ctx context.Context) string {
	// 尝试获取客户端IP作为键
	if clientIP, ok := ctx.Value("client_ip").(string); ok && clientIP != "" {
		return clientIP
	}
	return "global"
}

// defaultFailureHandler 默认的失败处理函数
func defaultFailureHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, errors.TooManyRequests("RATE_LIMITED", "too many requests")
}

// Limiter 限流器接口
type Limiter interface {
	// Allow 判断请求是否被允许，返回是否允许和剩余配额
	Allow(key string) (bool, int)
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	rate       float64            // 每秒填充的令牌数
	capacity   float64            // 桶的容量
	mutex      sync.Mutex         // 互斥锁
	buckets    map[string]*bucket // 桶映射表
	cleanupInt time.Duration      // 清理间隔
	lastClean  time.Time          // 上次清理时间
}

// bucket 令牌桶
type bucket struct {
	tokens     float64   // 当前令牌数
	lastUpdate time.Time // 上次更新时间
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(rate, capacity float64) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		rate:       rate,
		capacity:   capacity,
		buckets:    make(map[string]*bucket),
		cleanupInt: time.Minute * 10,
		lastClean:  time.Now(),
	}
	return limiter
}

// Allow 实现Limiter接口
func (l *TokenBucketLimiter) Allow(key string) (bool, int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()

	// 周期性清理不活跃的桶
	if now.Sub(l.lastClean) > l.cleanupInt {
		l.cleanup(now)
		l.lastClean = now
	}

	// 获取或创建桶
	b, exists := l.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     l.capacity,
			lastUpdate: now,
		}
		l.buckets[key] = b
	}

	// 添加新令牌（不超过桶容量）
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens = minFloat64(l.capacity, b.tokens+elapsed*l.rate)
	b.lastUpdate = now

	// 判断是否允许请求
	if b.tokens >= 1 {
		b.tokens--
		return true, int(b.tokens)
	}

	// 计算需要等待的时间（转换为毫秒）
	waitTime := int((1 - b.tokens) / l.rate * 1000)
	return false, waitTime
}

// cleanup 清理不活跃的桶
func (l *TokenBucketLimiter) cleanup(now time.Time) {
	// 清理超过30分钟不活跃的桶
	inactiveThreshold := now.Add(-30 * time.Minute)
	for key, b := range l.buckets {
		if b.lastUpdate.Before(inactiveThreshold) {
			delete(l.buckets, key)
		}
	}
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	rate       int                    // 允许的请求数
	window     time.Duration          // 窗口大小
	mutex      sync.Mutex             // 互斥锁
	windows    map[string]*timeWindow // 窗口映射表
	cleanupInt time.Duration          // 清理间隔
	lastClean  time.Time              // 上次清理时间
}

// timeWindow 时间窗口
type timeWindow struct {
	count      int       // 当前窗口计数
	timestamps []int64   // 请求时间戳
	lastUpdate time.Time // 上次更新时间
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(rate int, window time.Duration) *SlidingWindowLimiter {
	limiter := &SlidingWindowLimiter{
		rate:       rate,
		window:     window,
		windows:    make(map[string]*timeWindow),
		cleanupInt: time.Minute * 10,
		lastClean:  time.Now(),
	}
	return limiter
}

// Allow 实现Limiter接口
func (l *SlidingWindowLimiter) Allow(key string) (bool, int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()

	// 周期性清理不活跃的窗口
	if now.Sub(l.lastClean) > l.cleanupInt {
		l.cleanup(now)
		l.lastClean = now
	}

	// 获取或创建窗口
	w, exists := l.windows[key]
	if !exists {
		w = &timeWindow{
			count:      0,
			timestamps: make([]int64, 0, l.rate),
			lastUpdate: now,
		}
		l.windows[key] = w
	}

	// 移除窗口外的时间戳
	cutoff := now.Add(-l.window).UnixNano()
	validIdx := 0
	for i, ts := range w.timestamps {
		if ts >= cutoff {
			validIdx = i
			break
		}
	}
	if validIdx > 0 {
		w.timestamps = w.timestamps[validIdx:]
	}

	// 更新计数
	w.count = len(w.timestamps)
	w.lastUpdate = now

	// 判断是否允许请求
	if w.count < l.rate {
		w.timestamps = append(w.timestamps, now.UnixNano())
		w.count++
		return true, l.rate - w.count
	}

	// 计算需要等待的时间（转换为毫秒）
	waitTime := int(time.Duration(w.timestamps[0] - cutoff).Milliseconds())
	return false, waitTime
}

// cleanup 清理不活跃的窗口
func (l *SlidingWindowLimiter) cleanup(now time.Time) {
	// 清理超过30分钟不活跃的窗口
	inactiveThreshold := now.Add(-30 * time.Minute)
	for key, w := range l.windows {
		if w.lastUpdate.Before(inactiveThreshold) {
			delete(l.windows, key)
		}
	}
}

// GCRALimiter 一种适用于缓解流量突发的限流器
// GCRA (Generic Cell Rate Algorithm) 通用单元速率算法
type GCRALimiter struct {
	rate       float64              // 每秒允许的请求数
	burst      int                  // 允许的突发请求数
	mutex      sync.Mutex           // 互斥锁
	limiters   map[string]*gcraData // GCRA数据
	cleanupInt time.Duration        // 清理间隔
	lastClean  time.Time            // 上次清理时间
}

// gcraData GCRA算法数据
type gcraData struct {
	tau        time.Time // 理论到达时间
	lastUpdate time.Time // 上次更新时间
}

// NewGCRALimiter 创建GCRA限流器
func NewGCRALimiter(rate float64, burst int) *GCRALimiter {
	limiter := &GCRALimiter{
		rate:       rate,
		burst:      burst,
		limiters:   make(map[string]*gcraData),
		cleanupInt: time.Minute * 10,
		lastClean:  time.Now(),
	}
	return limiter
}

// Allow 实现Limiter接口
func (l *GCRALimiter) Allow(key string) (bool, int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()

	// 周期性清理不活跃的数据
	if now.Sub(l.lastClean) > l.cleanupInt {
		l.cleanup(now)
		l.lastClean = now
	}

	// 计算间隔时间
	increment := time.Duration(float64(time.Second) / l.rate)

	// 获取或创建数据
	data, exists := l.limiters[key]
	if !exists {
		data = &gcraData{
			tau:        now.Add(-time.Duration(l.burst) * increment),
			lastUpdate: now,
		}
		l.limiters[key] = data
	}

	// GCRA算法
	if data.tau.After(now) {
		// 计算需要等待的时间（毫秒）
		waitTime := int(data.tau.Sub(now).Milliseconds())
		return false, waitTime
	}

	// 更新理论到达时间
	data.tau = maxTime(data.tau.Add(increment), now)
	data.lastUpdate = now

	// 计算剩余配额
	remaining := int(now.Sub(data.tau).Milliseconds() / increment.Milliseconds())
	remaining = maxInt(0, minInt(l.burst, remaining))

	return true, remaining
}

// cleanup 清理不活跃的数据
func (l *GCRALimiter) cleanup(now time.Time) {
	// 清理超过30分钟不活跃的数据
	inactiveThreshold := now.Add(-30 * time.Minute)
	for key, data := range l.limiters {
		if data.lastUpdate.Before(inactiveThreshold) {
			delete(l.limiters, key)
		}
	}
}

// Limit 返回一个限流中间件
func Limit(opts ...Option) middleware.Middleware {
	// 默认选项
	options := options{
		logger:         log.DefaultLogger,
		keyFunc:        defaultKeyFunc,
		failureHandler: defaultFailureHandler,
	}
	for _, o := range opts {
		o(&options)
	}

	// 如果未提供限流器，使用默认的令牌桶限流器
	if options.limiter == nil {
		options.limiter = NewTokenBucketLimiter(100, 100)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 生成键
			key := options.keyFunc(ctx)

			// 检查限流
			allowed, remaining := options.limiter.Allow(key)
			if !allowed {
				options.logger.Warn("Rate limited",
					log.String("key", key),
					log.String("path", getPath(ctx)),
					log.Int("wait_ms", remaining),
				)
				return options.failureHandler(ctx, req)
			}

			// 记录限流信息
			if remaining < 10 {
				options.logger.Debug("Rate limit approaching",
					log.String("key", key),
					log.String("path", getPath(ctx)),
					log.Int("remaining", remaining),
				)
			}

			// 处理请求
			return handler(ctx, req)
		}
	}
}

// getPath 从上下文中获取路径
func getPath(ctx context.Context) string {
	if path, ok := ctx.Value("path").(string); ok {
		return path
	}
	return "unknown"
}

// 工具函数，避免命名冲突

// minFloat64 返回两个float64值中的较小值
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat64 返回两个float64值中的较大值
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// maxTime 返回两个时间中的较晚值
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// maxInt 返回两个int值中的较大值
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// minInt 返回两个int值中的较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
