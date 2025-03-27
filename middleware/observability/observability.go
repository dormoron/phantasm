package observability

import (
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
	"github.com/dormoron/phantasm/middleware/logging"
	"github.com/dormoron/phantasm/middleware/metrics"
	"github.com/dormoron/phantasm/middleware/tracing"
)

// Option 是可观测性中间件的选项
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

// WithMetrics 设置指标收集器
func WithMetrics(m metrics.Metrics) Option {
	return func(o *options) {
		o.metrics = m
	}
}

// WithTracer 设置跟踪器
func WithTracer(t tracing.Tracer) Option {
	return func(o *options) {
		o.tracer = t
	}
}

// options 是可观测性中间件的选项
type options struct {
	logger      log.Logger
	logReqBody  bool
	logRespBody bool
	metrics     metrics.Metrics
	tracer      tracing.Tracer
}

// Server 返回一个服务端可观测性中间件
// 组合了日志、指标和跟踪功能
func Server(opts ...Option) middleware.Middleware {
	options := options{
		logger:      log.DefaultLogger,
		logReqBody:  false,
		logRespBody: false,
	}
	for _, o := range opts {
		o(&options)
	}

	// 创建日志中间件
	loggingMiddleware := logging.Logging(
		logging.WithLogger(options.logger),
		logging.WithLogRequestBody(options.logReqBody),
		logging.WithLogResponseBody(options.logRespBody),
	)

	// 创建指标中间件
	metricsMiddleware := metrics.Server()
	if options.metrics != nil {
		metricsMiddleware = metrics.Server(
			metrics.WithMetrics(options.metrics),
		)
	}

	// 创建跟踪中间件
	tracingMiddleware := tracing.Server()
	if options.tracer != nil {
		tracingMiddleware = tracing.Server(
			tracing.WithTracer(options.tracer),
		)
	}

	// 组合中间件
	return middleware.Chain(
		loggingMiddleware,
		metricsMiddleware,
		tracingMiddleware,
	)
}

// Client 返回一个客户端可观测性中间件
// 组合了日志、指标和跟踪功能
func Client(opts ...Option) middleware.Middleware {
	options := options{
		logger:      log.DefaultLogger,
		logReqBody:  false,
		logRespBody: false,
	}
	for _, o := range opts {
		o(&options)
	}

	// 创建日志中间件
	loggingMiddleware := logging.Logging(
		logging.WithLogger(options.logger),
		logging.WithLogRequestBody(options.logReqBody),
		logging.WithLogResponseBody(options.logRespBody),
	)

	// 创建指标中间件
	metricsMiddleware := metrics.Client()
	if options.metrics != nil {
		metricsMiddleware = metrics.Client(
			metrics.WithMetrics(options.metrics),
		)
	}

	// 创建跟踪中间件
	tracingMiddleware := tracing.Client()
	if options.tracer != nil {
		tracingMiddleware = tracing.Client(
			tracing.WithTracer(options.tracer),
		)
	}

	// 组合中间件
	return middleware.Chain(
		loggingMiddleware,
		metricsMiddleware,
		tracingMiddleware,
	)
}
