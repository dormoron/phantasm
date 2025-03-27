package recovery

import (
	"context"
	"runtime"

	"github.com/dormoron/phantasm/errors"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是恢复中间件的选项
type Option func(*options)

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithHandler 设置恢复处理程序
func WithHandler(h Handler) Option {
	return func(o *options) {
		o.handler = h
	}
}

// options 是恢复中间件的选项
type options struct {
	logger  log.Logger
	handler Handler
}

// Handler 是恢复处理程序
type Handler func(ctx context.Context, req, err interface{}) error

// Recovery 返回一个恢复中间件
func Recovery(opts ...Option) middleware.Middleware {
	options := options{
		logger: log.DefaultLogger,
		handler: func(ctx context.Context, req, err interface{}) error {
			return errors.InternalServer("RECOVERY", "panic triggered")
		},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			defer func() {
				if rerr := recover(); rerr != nil {
					buf := make([]byte, 64<<10)
					n := runtime.Stack(buf, false)
					buf = buf[:n]
					options.logger.Error(
						"[RECOVERY]",
						log.Any("request", req),
						log.Any("error", rerr),
						log.String("stack", string(buf)),
					)
					err = options.handler(ctx, req, rerr)
				}
			}()
			return handler(ctx, req)
		}
	}
}
