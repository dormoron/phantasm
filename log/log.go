package log

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 是日志接口
type Logger interface {
	// Info 打印信息级别日志
	Info(msg string, fields ...Field)
	// Warn 打印警告级别日志
	Warn(msg string, fields ...Field)
	// Error 打印错误级别日志
	Error(msg string, fields ...Field)
	// Debug 打印调试级别日志
	Debug(msg string, fields ...Field)
	// WithContext 返回带有上下文的日志记录器
	WithContext(ctx context.Context) Logger
}

// Field 是日志字段
type Field struct {
	Key   string
	Value interface{}
}

// Option 是Logger配置选项
type Option func(*options)

// WithLevel 设置日志级别
func WithLevel(level Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithFormat 设置日志格式
func WithFormat(format Format) Option {
	return func(o *options) {
		o.format = format
	}
}

// WithOutput 设置日志输出
func WithOutput(output string) Option {
	return func(o *options) {
		o.output = output
	}
}

// WithCaller 设置是否打印调用者信息
func WithCaller(enabled bool) Option {
	return func(o *options) {
		o.caller = enabled
	}
}

// Level 表示日志级别
type Level int8

const (
	// DebugLevel 表示调试级别
	DebugLevel Level = iota - 1
	// InfoLevel 表示信息级别
	InfoLevel
	// WarnLevel 表示警告级别
	WarnLevel
	// ErrorLevel 表示错误级别
	ErrorLevel
	// FatalLevel 表示致命级别
	FatalLevel
)

// Format 表示日志格式
type Format string

const (
	// TextFormat 表示文本格式
	TextFormat Format = "text"
	// JSONFormat 表示JSON格式
	JSONFormat Format = "json"
)

// DefaultLogger 是默认的日志记录器
var DefaultLogger Logger = NewStdLogger()

var (
	once   sync.Once
	global Logger
)

// options 是Logger配置
type options struct {
	level  Level
	format Format
	output string
	caller bool
}

// defaultOptions 返回默认配置
func defaultOptions() *options {
	return &options{
		level:  InfoLevel,
		format: TextFormat,
		output: "stdout",
		caller: true,
	}
}

// New 创建一个日志记录器
func New(opts ...Option) Logger {
	options := defaultOptions()
	for _, o := range opts {
		o(options)
	}
	return NewZapLogger(options)
}

// NewZapLogger 创建基于zap的日志记录器
func NewZapLogger(opts *options) Logger {
	var zapLevel zapcore.Level
	switch opts.level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case FatalLevel:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if opts.format == JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var output zapcore.WriteSyncer
	switch opts.output {
	case "stdout":
		output = zapcore.AddSync(os.Stdout)
	case "stderr":
		output = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(opts.output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		output = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, output, zapLevel)

	var zapOpts []zap.Option
	if opts.caller {
		zapOpts = append(zapOpts, zap.AddCaller(), zap.AddCallerSkip(2))
	}

	logger := zap.New(core, zapOpts...)
	return &zapLogger{
		logger: logger,
	}
}

// zapLogger 是基于zap的日志记录器
type zapLogger struct {
	logger *zap.Logger
}

// Info 打印信息级别日志
func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.convertFields(fields...)...)
}

// Warn 打印警告级别日志
func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.convertFields(fields...)...)
}

// Error 打印错误级别日志
func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, l.convertFields(fields...)...)
}

// Debug 打印调试级别日志
func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.convertFields(fields...)...)
}

// WithContext 返回带有上下文的日志记录器
func (l *zapLogger) WithContext(ctx context.Context) Logger {
	return l
}

// convertFields 将Field转换为zap.Field
func (l *zapLogger) convertFields(fields ...Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	return zapFields
}

// Err 创建一个错误字段
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// String 创建一个字符串字段
func String(key string, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建一个整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建一个布尔字段
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Any 创建一个任意类型字段
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建一个浮点数字段
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// SetLogger 设置全局日志记录器
func SetLogger(logger Logger) {
	once.Do(func() {
		global = logger
		DefaultLogger = logger
	})
}

// Info 打印信息级别日志
func Info(msg string, fields ...Field) {
	DefaultLogger.Info(msg, fields...)
}

// Warn 打印警告级别日志
func Warn(msg string, fields ...Field) {
	DefaultLogger.Warn(msg, fields...)
}

// Error 打印错误级别日志
func Error(msg string, fields ...Field) {
	DefaultLogger.Error(msg, fields...)
}

// Debug 打印调试级别日志
func Debug(msg string, fields ...Field) {
	DefaultLogger.Debug(msg, fields...)
}

// WithContext 返回带有上下文的日志记录器
func WithContext(ctx context.Context) Logger {
	return DefaultLogger.WithContext(ctx)
}

// simpleFormatMessage 格式化日志消息
func simpleFormatMessage(msg string, fields ...Field) string {
	if len(fields) == 0 {
		return msg
	}
	result := msg
	for _, field := range fields {
		result = fmt.Sprintf("%s %s=%v", result, field.Key, field.Value)
	}
	return result
}

// stdLogger 是基于标准库的日志记录器
type stdLogger struct{}

// NewStdLogger 创建一个标准日志记录器
func NewStdLogger() Logger {
	return &stdLogger{}
}

// Info 打印信息级别日志
func (l *stdLogger) Info(msg string, fields ...Field) {
	fmt.Println("[INFO]", simpleFormatMessage(msg, fields...))
}

// Warn 打印警告级别日志
func (l *stdLogger) Warn(msg string, fields ...Field) {
	fmt.Println("[WARN]", simpleFormatMessage(msg, fields...))
}

// Error 打印错误级别日志
func (l *stdLogger) Error(msg string, fields ...Field) {
	fmt.Println("[ERROR]", simpleFormatMessage(msg, fields...))
}

// Debug 打印调试级别日志
func (l *stdLogger) Debug(msg string, fields ...Field) {
	fmt.Println("[DEBUG]", simpleFormatMessage(msg, fields...))
}

// WithContext 返回带有上下文的日志记录器
func (l *stdLogger) WithContext(ctx context.Context) Logger {
	return l
}
