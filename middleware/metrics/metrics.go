package metrics

import (
	"context"
	"time"

	"github.com/dormoron/phantasm/middleware"
)

// Option 是指标中间件的选项
type Option func(*options)

// Metrics 是指标收集接口
type Metrics interface {
	// Counter 计数器
	Counter(name string, value float64, labels map[string]string)
	// Gauge 仪表
	Gauge(name string, value float64, labels map[string]string)
	// Histogram 直方图
	Histogram(name string, value float64, labels map[string]string)

	// CreateCounter 创建计数器
	CreateCounter(opts CounterOptions) Counter
	// CreateCounterVec 创建带标签的计数器
	CreateCounterVec(opts CounterOptions, labelNames []string) CounterVec
	// CreateGauge 创建仪表盘
	CreateGauge(opts GaugeOptions) Gauge
	// CreateGaugeVec 创建带标签的仪表盘
	CreateGaugeVec(opts GaugeOptions, labelNames []string) GaugeVec
	// CreateHistogram 创建直方图
	CreateHistogram(opts HistogramOptions) Histogram
	// CreateHistogramVec 创建带标签的直方图
	CreateHistogramVec(opts HistogramOptions, labelNames []string) HistogramVec
	// CreateSummary 创建摘要
	CreateSummary(opts SummaryOptions) Summary
	// CreateSummaryVec 创建带标签的摘要
	CreateSummaryVec(opts SummaryOptions, labelNames []string) SummaryVec
}

// CounterOptions 计数器选项
type CounterOptions struct {
	Name   string            // 指标名称
	Help   string            // 指标帮助信息
	Labels map[string]string // 指标标签
}

// Counter 计数器接口
type Counter interface {
	// Inc 增加1
	Inc()
	// Add 增加给定值
	Add(float64)
}

// CounterVec 带标签的计数器接口
type CounterVec interface {
	// With 使用标签获取Counter
	With(map[string]string) Counter
}

// GaugeOptions 仪表盘选项
type GaugeOptions struct {
	Name   string            // 指标名称
	Help   string            // 指标帮助信息
	Labels map[string]string // 指标标签
}

// Gauge 仪表盘接口
type Gauge interface {
	// Set 设置值
	Set(float64)
	// Inc 增加1
	Inc()
	// Dec 减少1
	Dec()
	// Add 增加给定值
	Add(float64)
	// Sub 减少给定值
	Sub(float64)
	// SetToCurrentTime 设置为当前时间戳
	SetToCurrentTime()
}

// GaugeVec 带标签的仪表盘接口
type GaugeVec interface {
	// With 使用标签获取Gauge
	With(map[string]string) Gauge
}

// HistogramOptions 直方图选项
type HistogramOptions struct {
	Name    string            // 指标名称
	Help    string            // 指标帮助信息
	Labels  map[string]string // 指标标签
	Buckets []float64         // 桶数组
}

// Histogram 直方图接口
type Histogram interface {
	// Observe 观察值
	Observe(float64)
}

// HistogramVec 带标签的直方图接口
type HistogramVec interface {
	// With 使用标签获取Histogram
	With(map[string]string) Histogram
}

// SummaryOptions 摘要选项
type SummaryOptions struct {
	Name       string              // 指标名称
	Help       string              // 指标帮助信息
	Labels     map[string]string   // 指标标签
	Objectives map[float64]float64 // 分位数
	MaxAge     time.Duration       // 最大存活时间
	AgeBuckets uint32              // 年龄桶数量
	BufCap     uint32              // 缓冲区容量
}

// Summary 摘要接口
type Summary interface {
	// Observe 观察值
	Observe(float64)
}

// SummaryVec 带标签的摘要接口
type SummaryVec interface {
	// With 使用标签获取Summary
	With(map[string]string) Summary
}

// WithMetrics 设置指标收集器
func WithMetrics(metrics Metrics) Option {
	return func(o *options) {
		o.metrics = metrics
	}
}

// options 是指标中间件的选项
type options struct {
	metrics Metrics
}

// defaultMetrics 是默认的指标收集器，不执行任何操作
type defaultMetrics struct{}

// Counter 实现Metrics接口
func (m defaultMetrics) Counter(name string, value float64, labels map[string]string) {}

// Gauge 实现Metrics接口
func (m defaultMetrics) Gauge(name string, value float64, labels map[string]string) {}

// Histogram 实现Metrics接口
func (m defaultMetrics) Histogram(name string, value float64, labels map[string]string) {}

// CreateCounter 实现Metrics接口
func (m defaultMetrics) CreateCounter(opts CounterOptions) Counter {
	return &defaultCounter{}
}

// CreateCounterVec 实现Metrics接口
func (m defaultMetrics) CreateCounterVec(opts CounterOptions, labelNames []string) CounterVec {
	return &defaultCounterVec{}
}

// CreateGauge 实现Metrics接口
func (m defaultMetrics) CreateGauge(opts GaugeOptions) Gauge {
	return &defaultGauge{}
}

// CreateGaugeVec 实现Metrics接口
func (m defaultMetrics) CreateGaugeVec(opts GaugeOptions, labelNames []string) GaugeVec {
	return &defaultGaugeVec{}
}

// CreateHistogram 实现Metrics接口
func (m defaultMetrics) CreateHistogram(opts HistogramOptions) Histogram {
	return &defaultHistogram{}
}

// CreateHistogramVec 实现Metrics接口
func (m defaultMetrics) CreateHistogramVec(opts HistogramOptions, labelNames []string) HistogramVec {
	return &defaultHistogramVec{}
}

// CreateSummary 实现Metrics接口
func (m defaultMetrics) CreateSummary(opts SummaryOptions) Summary {
	return &defaultSummary{}
}

// CreateSummaryVec 实现Metrics接口
func (m defaultMetrics) CreateSummaryVec(opts SummaryOptions, labelNames []string) SummaryVec {
	return &defaultSummaryVec{}
}

// defaultCounter 是默认的计数器实现
type defaultCounter struct{}

// Inc 实现Counter接口
func (c *defaultCounter) Inc() {}

// Add 实现Counter接口
func (c *defaultCounter) Add(float64) {}

// defaultCounterVec 是默认的带标签计数器实现
type defaultCounterVec struct{}

// With 实现CounterVec接口
func (cv *defaultCounterVec) With(map[string]string) Counter {
	return &defaultCounter{}
}

// defaultGauge 是默认的仪表盘实现
type defaultGauge struct{}

// Set 实现Gauge接口
func (g *defaultGauge) Set(float64) {}

// Inc 实现Gauge接口
func (g *defaultGauge) Inc() {}

// Dec 实现Gauge接口
func (g *defaultGauge) Dec() {}

// Add 实现Gauge接口
func (g *defaultGauge) Add(float64) {}

// Sub 实现Gauge接口
func (g *defaultGauge) Sub(float64) {}

// SetToCurrentTime 实现Gauge接口
func (g *defaultGauge) SetToCurrentTime() {}

// defaultGaugeVec 是默认的带标签仪表盘实现
type defaultGaugeVec struct{}

// With 实现GaugeVec接口
func (gv *defaultGaugeVec) With(map[string]string) Gauge {
	return &defaultGauge{}
}

// defaultHistogram 是默认的直方图实现
type defaultHistogram struct{}

// Observe 实现Histogram接口
func (h *defaultHistogram) Observe(float64) {}

// defaultHistogramVec 是默认的带标签直方图实现
type defaultHistogramVec struct{}

// With 实现HistogramVec接口
func (hv *defaultHistogramVec) With(map[string]string) Histogram {
	return &defaultHistogram{}
}

// defaultSummary 是默认的摘要实现
type defaultSummary struct{}

// Observe 实现Summary接口
func (s *defaultSummary) Observe(float64) {}

// defaultSummaryVec 是默认的带标签摘要实现
type defaultSummaryVec struct{}

// With 实现SummaryVec接口
func (sv *defaultSummaryVec) With(map[string]string) Summary {
	return &defaultSummary{}
}

// Server 返回一个服务端指标中间件
func Server(opts ...Option) middleware.Middleware {
	options := options{
		metrics: defaultMetrics{},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			startTime := time.Now()

			// 记录请求计数
			options.metrics.Counter("request_total", 1, map[string]string{
				"path":   getPath(ctx),
				"method": getMethod(ctx),
			})

			// 处理请求
			resp, err := handler(ctx, req)

			// 记录请求时长
			duration := time.Since(startTime).Seconds()
			options.metrics.Histogram("request_duration_seconds", duration, map[string]string{
				"path":   getPath(ctx),
				"method": getMethod(ctx),
			})

			// 记录错误计数
			if err != nil {
				options.metrics.Counter("request_error_total", 1, map[string]string{
					"path":   getPath(ctx),
					"method": getMethod(ctx),
					"err":    err.Error(),
				})
			}

			return resp, err
		}
	}
}

// Client 返回一个客户端指标中间件
func Client(opts ...Option) middleware.Middleware {
	options := options{
		metrics: defaultMetrics{},
	}
	for _, o := range opts {
		o(&options)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			startTime := time.Now()

			// 记录请求计数
			options.metrics.Counter("client_request_total", 1, map[string]string{
				"path":   getPath(ctx),
				"method": getMethod(ctx),
			})

			// 处理请求
			resp, err := handler(ctx, req)

			// 记录请求时长
			duration := time.Since(startTime).Seconds()
			options.metrics.Histogram("client_request_duration_seconds", duration, map[string]string{
				"path":   getPath(ctx),
				"method": getMethod(ctx),
			})

			// 记录错误计数
			if err != nil {
				options.metrics.Counter("client_request_error_total", 1, map[string]string{
					"path":   getPath(ctx),
					"method": getMethod(ctx),
					"err":    err.Error(),
				})
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
