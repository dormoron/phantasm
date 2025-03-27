package jaeger

import (
	"context"
	"fmt"
	"time"

	"github.com/dormoron/phantasm/middleware/tracing"

	"github.com/dormoron/phantasm/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// Option Jaeger配置选项
type Option func(*options)

// options Jaeger配置
type options struct {
	Endpoint string        // Jaeger endpoint，例如 http://localhost:14268/api/traces
	Name     string        // 服务名称
	Version  string        // 服务版本
	Env      string        // 环境名称
	Timeout  time.Duration // 超时时间
	Logger   log.Logger    // 日志接口
}

// WithEndpoint 设置Jaeger endpoint
func WithEndpoint(endpoint string) Option {
	return func(o *options) {
		o.Endpoint = endpoint
	}
}

// WithServiceName 设置服务名称
func WithServiceName(name string) Option {
	return func(o *options) {
		o.Name = name
	}
}

// WithServiceVersion 设置服务版本
func WithServiceVersion(version string) Option {
	return func(o *options) {
		o.Version = version
	}
}

// WithEnvironment 设置环境名称
func WithEnvironment(env string) Option {
	return func(o *options) {
		o.Env = env
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.Timeout = timeout
	}
}

// WithLogger 设置日志
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.Logger = logger
	}
}

// NewTracer 创建一个基于Jaeger的Tracer
func NewTracer(opts ...Option) (*Tracer, error) {
	options := &options{
		Endpoint: "http://localhost:14268/api/traces",
		Name:     "phantasm-service",
		Version:  "unknown",
		Env:      "development",
		Timeout:  time.Second * 5,
		Logger:   log.DefaultLogger,
	}

	for _, o := range opts {
		o(options)
	}

	// 创建Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(options.Endpoint)))
	if err != nil {
		return nil, err
	}

	// 创建资源
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(options.Name),
			semconv.ServiceVersionKey.String(options.Version),
			attribute.String("environment", options.Env),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建tracer provider
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp, tracesdk.WithMaxExportBatchSize(1)),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)

	// 设置全局propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 设置全局tracer provider
	otel.SetTracerProvider(provider)

	return &Tracer{
		provider: provider,
		tracer:   provider.Tracer(options.Name),
		options:  options,
	}, nil
}

// Tracer 基于Jaeger的分布式追踪实现
type Tracer struct {
	provider *tracesdk.TracerProvider
	tracer   trace.Tracer
	options  *options
}

// Start 开始一个span
func (t *Tracer) Start(ctx context.Context, operation string) (context.Context, tracing.Span) {
	ctx, span := t.tracer.Start(ctx, operation)
	return ctx, &Span{span: span}
}

// Close 关闭tracer
func (t *Tracer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), t.options.Timeout)
	defer cancel()
	return t.provider.Shutdown(ctx)
}

// Extract 从context中提取span上下文
func (t *Tracer) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// Inject 将span上下文注入到carrier
func (t *Tracer) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// Span 封装OpenTelemetry的Span
type Span struct {
	span trace.Span
}

// SetAttributes 设置span属性
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	s.span.SetAttributes(attrs...)
}

// AddEvent 添加事件
func (s *Span) AddEvent(name string, attrs ...attribute.KeyValue) {
	s.span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetStatus 设置状态
func (s *Span) SetStatus(code tracing.StatusCode, description string) {
	s.span.SetStatus(codes.Code(code), description)
}

// RecordError 记录错误
func (s *Span) RecordError(err error, opts ...trace.EventOption) {
	s.span.RecordError(err, opts...)
}

// SetTag 设置Span标签，实现tracing.Span接口
func (s *Span) SetTag(key string, value interface{}) {
	var attr attribute.KeyValue
	switch v := value.(type) {
	case string:
		attr = attribute.String(key, v)
	case bool:
		attr = attribute.Bool(key, v)
	case int:
		attr = attribute.Int(key, v)
	case int64:
		attr = attribute.Int64(key, v)
	case float64:
		attr = attribute.Float64(key, v)
	default:
		// 尝试将其他类型转为字符串
		attr = attribute.String(key, fmt.Sprintf("%v", v))
	}
	s.span.SetAttributes(attr)
}

// SetError 设置Span错误，实现tracing.Span接口
func (s *Span) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// End 结束span
func (s *Span) End() {
	s.span.End()
}

// SpanFromContext 从上下文获取span
func SpanFromContext(ctx context.Context) tracing.Span {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		// 返回一个无操作的span
		return &noopSpan{}
	}
	return &Span{span: span}
}

// NewNoopTracer 创建无操作的Tracer
func NewNoopTracer() tracing.Tracer {
	return &noopTracer{}
}

// noopTracer 无操作的tracer实现
type noopTracer struct{}

// Start 开始一个无操作的span
func (t *noopTracer) Start(ctx context.Context, name string) (context.Context, tracing.Span) {
	return ctx, &noopSpan{}
}

// Close 关闭tracer
func (t *noopTracer) Close() error {
	return nil
}

// Extract 从context中提取span上下文
func (t *noopTracer) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return ctx
}

// Inject 将span上下文注入到carrier
func (t *noopTracer) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	// 无操作
}

// noopSpan 无操作的span实现
type noopSpan struct{}

// SetAttributes 设置span属性
func (s *noopSpan) SetAttributes(attrs ...attribute.KeyValue) {
	// 无操作
}

// AddEvent 添加事件
func (s *noopSpan) AddEvent(name string, attrs ...attribute.KeyValue) {
	// 无操作
}

// SetStatus 设置状态
func (s *noopSpan) SetStatus(code tracing.StatusCode, description string) {
	// 无操作
}

// RecordError 记录错误
func (s *noopSpan) RecordError(err error, opts ...trace.EventOption) {
	// 无操作
}

// SetTag 设置Span标签，实现tracing.Span接口
func (s *noopSpan) SetTag(key string, value interface{}) {
	// 无操作
}

// SetError 设置Span错误，实现tracing.Span接口
func (s *noopSpan) SetError(err error) {
	// 无操作
}

// End 结束Span
func (s *noopSpan) End() {
	// 无操作
}
