package phantasm

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/registry"
	"github.com/dormoron/phantasm/transport"
)

// Option 是应用程序选项的函数类型
type Option func(o *options)

// options 是应用程序选项
type options struct {
	id               string
	name             string
	version          string
	metadata         map[string]string
	endpoints        []*url.URL
	ctx              context.Context
	sigs             []os.Signal
	registrar        registry.Registrar
	registrarTimeout time.Duration
	stopTimeout      time.Duration
	logger           log.Logger
	servers          []transport.Server
	beforeStart      []func(context.Context) error
	afterStart       []func(context.Context) error
	beforeStop       []func(context.Context) error
	afterStop        []func(context.Context) error
}

// ID 设置应用程序的实例ID
func ID(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

// Name 设置应用程序的名称
func Name(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// Version 设置应用程序的版本
func Version(version string) Option {
	return func(o *options) {
		o.version = version
	}
}

// Metadata 设置应用程序的元数据
func Metadata(md map[string]string) Option {
	return func(o *options) {
		o.metadata = md
	}
}

// Endpoint 添加应用程序的端点
func Endpoint(endpoints ...*url.URL) Option {
	return func(o *options) {
		o.endpoints = append(o.endpoints, endpoints...)
	}
}

// Context 设置应用程序的上下文
func Context(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
	}
}

// Signal 设置应用程序的信号
func Signal(sigs ...os.Signal) Option {
	return func(o *options) {
		o.sigs = sigs
	}
}

// Registrar 设置应用程序的注册器
func Registrar(r registry.Registrar) Option {
	return func(o *options) {
		o.registrar = r
	}
}

// RegistrarTimeout 设置应用程序的注册超时
func RegistrarTimeout(t time.Duration) Option {
	return func(o *options) {
		o.registrarTimeout = t
	}
}

// StopTimeout 设置应用程序的停止超时
func StopTimeout(t time.Duration) Option {
	return func(o *options) {
		o.stopTimeout = t
	}
}

// Logger 设置应用程序的日志记录器
func Logger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// Server 添加应用程序的服务器
func Server(srv ...transport.Server) Option {
	return func(o *options) {
		o.servers = append(o.servers, srv...)
	}
}

// BeforeStart 添加在应用程序启动前执行的函数
func BeforeStart(fn func(context.Context) error) Option {
	return func(o *options) {
		o.beforeStart = append(o.beforeStart, fn)
	}
}

// AfterStart 添加在应用程序启动后执行的函数
func AfterStart(fn func(context.Context) error) Option {
	return func(o *options) {
		o.afterStart = append(o.afterStart, fn)
	}
}

// BeforeStop 添加在应用程序停止前执行的函数
func BeforeStop(fn func(context.Context) error) Option {
	return func(o *options) {
		o.beforeStop = append(o.beforeStop, fn)
	}
}

// AfterStop 添加在应用程序停止后执行的函数
func AfterStop(fn func(context.Context) error) Option {
	return func(o *options) {
		o.afterStop = append(o.afterStop, fn)
	}
}
