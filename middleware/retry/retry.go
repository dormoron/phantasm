package retry

import (
	"context"
	"time"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是重试中间件的选项
type Option func(*options)

// WithAttempts 设置最大重试次数（包括首次尝试）
func WithAttempts(attempts int) Option {
	return func(o *options) {
		o.attempts = attempts
	}
}

// WithDelay 设置重试间隔
func WithDelay(delay time.Duration) Option {
	return func(o *options) {
		o.delay = delay
	}
}

// WithBackoff 设置指数退避策略
func WithBackoff(backoff bool) Option {
	return func(o *options) {
		o.backoff = backoff
	}
}

// WithMaxDelay 设置最大重试延迟
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(o *options) {
		o.maxDelay = maxDelay
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// RetryIfFunc 定义重试判断函数类型
type RetryIfFunc func(ctx context.Context, req, resp interface{}, err error) bool

// WithRetryIf 设置重试条件
func WithRetryIf(fn RetryIfFunc) Option {
	return func(o *options) {
		o.retryIf = fn
	}
}

// options 是重试中间件的选项
type options struct {
	attempts int
	delay    time.Duration
	backoff  bool
	maxDelay time.Duration
	logger   log.Logger
	retryIf  RetryIfFunc
}

// defaultRetryIf 默认的重试条件：只重试非业务错误
func defaultRetryIf(ctx context.Context, req, resp interface{}, err error) bool {
	if err == nil {
		return false
	}

	// 检查是否为业务错误
	if e, ok := err.(*errors.Error); ok {
		// 不重试客户端错误（4xx）
		if e.Code >= 400 && e.Code < 500 {
			return false
		}
	}

	// 默认重试所有其他错误
	return true
}

// Retry 返回一个重试中间件，主要用于客户端请求
func Retry(opts ...Option) middleware.Middleware {
	options := options{
		attempts: 3,
		delay:    time.Millisecond * 100,
		backoff:  true,
		maxDelay: time.Second * 2,
		logger:   log.DefaultLogger,
		retryIf:  defaultRetryIf,
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var (
				resp    interface{}
				err     error
				attempt int
				delay   = options.delay
			)

			for attempt = 0; attempt < options.attempts; attempt++ {
				resp, err = handler(ctx, req)

				// 检查是否需要重试
				if !options.retryIf(ctx, req, resp, err) {
					break
				}

				// 最后一次尝试，不再重试
				if attempt == options.attempts-1 {
					break
				}

				// 计算延迟时间
				if options.backoff && attempt > 0 {
					delay = time.Duration(float64(delay) * 1.5)
					if delay > options.maxDelay {
						delay = options.maxDelay
					}
				}

				// 记录重试信息
				options.logger.Info(
					"Retrying request",
					log.Int("attempt", attempt+1),
					log.Int("max_attempts", options.attempts),
					log.String("delay", delay.String()),
					log.String("path", getPath(ctx)),
					log.Any("error", err),
				)

				// 等待一段时间再重试
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
					// 继续重试
				}
			}

			if err != nil && attempt > 0 {
				// 记录重试失败信息
				options.logger.Error(
					"Retry failed",
					log.Int("attempts", attempt+1),
					log.String("path", getPath(ctx)),
					log.Any("error", err),
				)
			}

			return resp, err
		}
	}
}

// getPath 从上下文中获取请求路径
func getPath(ctx context.Context) string {
	if path, ok := ctx.Value("path").(string); ok {
		return path
	}
	return "unknown"
}
