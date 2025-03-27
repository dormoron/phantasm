package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是日志中间件的选项
type Option func(*options)

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithLogRequestBody 设置是否记录请求体
func WithLogRequestBody(enabled bool) Option {
	return func(o *options) {
		o.logReqBody = enabled
	}
}

// WithLogResponseBody 设置是否记录响应体
func WithLogResponseBody(enabled bool) Option {
	return func(o *options) {
		o.logRespBody = enabled
	}
}

// WithSkipper 添加跳过日志记录的条件
func WithSkipper(skipper Skipper) Option {
	return func(o *options) {
		o.skipper = skipper
	}
}

// WithMaxBodyLogSize 限制记录的请求/响应体大小
func WithMaxBodyLogSize(size int) Option {
	return func(o *options) {
		o.maxBodyLogSize = size
	}
}

// Skipper 定义哪些请求应该跳过日志记录
type Skipper func(ctx context.Context, req interface{}) bool

// options 是日志中间件的选项
type options struct {
	logger         log.Logger
	logReqBody     bool
	logRespBody    bool
	skipper        Skipper
	maxBodyLogSize int
}

// Logging 返回一个日志中间件
func Logging(opts ...Option) middleware.Middleware {
	options := options{
		logger:         log.DefaultLogger,
		logReqBody:     false,
		logRespBody:    false,
		maxBodyLogSize: 4096, // 默认限制4KB
		skipper: func(ctx context.Context, req interface{}) bool {
			return false // 默认不跳过任何请求
		},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// 检查是否需要跳过日志记录
			if options.skipper(ctx, req) {
				return handler(ctx, req)
			}

			startTime := time.Now()
			reqID := GetRequestID(ctx)

			// 记录请求
			fields := []log.Field{
				log.String("path", getPath(ctx)),
				log.String("method", getMethod(ctx)),
				log.String("request_id", reqID),
			}

			// 添加客户端信息
			if clientIP := getClientIP(ctx); clientIP != "" {
				fields = append(fields, log.String("client_ip", clientIP))
			}

			// 添加请求体
			if options.logReqBody {
				reqBody := formatBody(req, options.maxBodyLogSize)
				fields = append(fields, log.Any("request", reqBody))
			}

			options.logger.Info("Request", fields...)

			// 处理请求
			resp, err := handler(ctx, req)

			// 记录响应
			duration := time.Since(startTime)
			respFields := []log.Field{
				log.String("path", getPath(ctx)),
				log.String("method", getMethod(ctx)),
				log.String("request_id", reqID),
				log.Float64("duration_ms", float64(duration.Milliseconds())),
			}

			// 添加响应体
			if options.logRespBody && resp != nil {
				respBody := formatBody(resp, options.maxBodyLogSize)
				respFields = append(respFields, log.Any("response", respBody))
			}

			// 记录错误
			if err != nil {
				respFields = append(respFields, log.Any("error", err))
				options.logger.Error("Response", respFields...)
			} else {
				options.logger.Info("Response", respFields...)
			}

			return resp, err
		}
	}
}

// GetRequestID 从上下文中获取请求ID
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value("request_id").(string); ok && id != "" {
		return id
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
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

// getClientIP 从上下文中获取客户端IP
func getClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip
	}
	return ""
}

// formatBody 格式化请求/响应体，限制大小
func formatBody(body interface{}, maxSize int) interface{} {
	if body == nil {
		return nil
	}

	// 对于字符串类型，我们可以截断
	if str, ok := body.(string); ok && maxSize > 0 && len(str) > maxSize {
		return str[:maxSize] + "...(truncated)"
	}

	return body
}
