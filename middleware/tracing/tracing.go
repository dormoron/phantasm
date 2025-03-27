package tracing

import (
	"context"

	"github.com/dormoron/phantasm/middleware"
)

// StatusCode 表示追踪状态码
type StatusCode int

const (
	// StatusOk 表示操作成功
	StatusOk StatusCode = 0
	// StatusError 表示操作失败
	StatusError StatusCode = 1
)

// Option 是跟踪中间件的选项
type Option func(*options)

// Tracer 是跟踪器接口
type Tracer interface {
	// Start 开始一个新的跟踪Span
	Start(ctx context.Context, operation string) (context.Context, Span)
}

// Span 是跟踪Span接口
type Span interface {
	// End 结束Span
	End()
	// SetTag 设置Span标签
	SetTag(key string, value interface{})
	// SetError 设置Span错误
	SetError(err error)
	// SetStatus 设置状态
	SetStatus(code StatusCode, description string)
}

// WithTracer 设置跟踪器
func WithTracer(tracer Tracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}

// options 是跟踪中间件的选项
type options struct {
	tracer Tracer
}

// defaultTracer 是默认的跟踪器，不执行任何操作
type defaultTracer struct{}

// Start 实现Tracer接口
func (t defaultTracer) Start(ctx context.Context, operation string) (context.Context, Span) {
	return ctx, defaultSpan{}
}

// defaultSpan 是默认的Span，不执行任何操作
type defaultSpan struct{}

// End 实现Span接口
func (s defaultSpan) End() {}

// SetTag 实现Span接口
func (s defaultSpan) SetTag(key string, value interface{}) {}

// SetError 实现Span接口
func (s defaultSpan) SetError(err error) {}

// SetStatus 实现Span接口
func (s defaultSpan) SetStatus(code StatusCode, description string) {}

// Server 返回一个服务端跟踪中间件
func Server(opts ...Option) middleware.Middleware {
	options := options{
		tracer: defaultTracer{},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 创建操作名称
			operation := getMethod(ctx) + " " + getPath(ctx)

			// 开始跟踪
			ctx, span := options.tracer.Start(ctx, operation)
			defer span.End()

			// 设置标签
			span.SetTag("component", "server")
			span.SetTag("path", getPath(ctx))
			span.SetTag("method", getMethod(ctx))

			// 处理请求
			resp, err := handler(ctx, req)

			// 设置错误
			if err != nil {
				span.SetError(err)
				span.SetTag("error", true)
			}

			return resp, err
		}
	}
}

// Client 返回一个客户端跟踪中间件
func Client(opts ...Option) middleware.Middleware {
	options := options{
		tracer: defaultTracer{},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 创建操作名称
			operation := "client " + getMethod(ctx) + " " + getPath(ctx)

			// 开始跟踪
			ctx, span := options.tracer.Start(ctx, operation)
			defer span.End()

			// 设置标签
			span.SetTag("component", "client")
			span.SetTag("path", getPath(ctx))
			span.SetTag("method", getMethod(ctx))

			// 处理请求
			resp, err := handler(ctx, req)

			// 设置错误
			if err != nil {
				span.SetError(err)
				span.SetTag("error", true)
			}

			return resp, err
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

// getMethod 从上下文中获取方法
func getMethod(ctx context.Context) string {
	if method, ok := ctx.Value("method").(string); ok {
		return method
	}
	return "unknown"
}
