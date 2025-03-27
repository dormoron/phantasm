package phantasm

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/dormoron/phantasm/internal/server"
	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/registry"
	"github.com/dormoron/phantasm/transport"
)

// App 是应用程序接口
type App interface {
	// ID 返回应用程序ID
	ID() string
	// Name 返回应用程序名称
	Name() string
	// Version 返回应用程序版本
	Version() string
	// Metadata 返回应用程序元数据
	Metadata() map[string]string
	// Start 启动应用程序
	Start() error
	// Stop 停止应用程序
	Stop() error
}

// 注意：Option类型和选项函数在options.go中定义

// appOptions 是app.go内部使用的应用程序选项，兼容options.go中的选项
type appOptions struct {
	id          string
	name        string
	version     string
	metadata    map[string]string
	servers     []transport.Server
	registrar   registry.Registrar
	ctx         context.Context
	sigs        []os.Signal
	stopTimeout time.Duration
	logger      log.Logger
}

// application 是应用程序实现
type application struct {
	opts          appOptions
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	log           log.Logger
	serverManager server.Manager
}

// New 创建一个新的应用程序
func New(opts ...Option) App {
	// 创建内部使用的选项
	o := appOptions{
		ctx:         context.Background(),
		sigs:        []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT},
		stopTimeout: time.Second * 30,
		metadata:    make(map[string]string),
	}

	// 创建用于接收options.go中选项的临时结构
	externalOpts := options{
		ctx:         context.Background(),
		sigs:        []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT},
		stopTimeout: time.Second * 30,
		metadata:    make(map[string]string),
	}

	// 应用外部选项
	for _, opt := range opts {
		opt(&externalOpts)
	}

	// 映射到内部选项结构
	o.id = externalOpts.id
	o.name = externalOpts.name
	o.version = externalOpts.version
	o.metadata = externalOpts.metadata
	o.servers = externalOpts.servers
	if externalOpts.registrar != nil {
		o.registrar = externalOpts.registrar
	}
	o.ctx = externalOpts.ctx
	o.sigs = externalOpts.sigs
	o.stopTimeout = externalOpts.stopTimeout
	o.logger = externalOpts.logger

	ctx, cancel := context.WithCancel(o.ctx)
	logger := o.logger
	if logger == nil {
		logger = log.DefaultLogger
	}

	// 创建服务器管理器
	serverManager := server.NewManager(logger)

	return &application{
		opts:          o,
		ctx:           ctx,
		cancel:        cancel,
		log:           logger,
		serverManager: serverManager,
	}
}

// ID 返回应用程序ID
func (a *application) ID() string {
	return a.opts.id
}

// Name 返回应用程序名称
func (a *application) Name() string {
	return a.opts.name
}

// Version 返回应用程序版本
func (a *application) Version() string {
	return a.opts.version
}

// Metadata 返回应用程序元数据
func (a *application) Metadata() map[string]string {
	return a.opts.metadata
}

// Start 启动应用程序
func (a *application) Start() error {
	a.mu.Lock()
	a.log.Info("启动应用程序", log.String("id", a.opts.id), log.String("name", a.opts.name), log.String("version", a.opts.version))

	// 初始化服务器管理器
	for _, srv := range a.opts.servers {
		a.serverManager.Add(srv)
	}
	a.mu.Unlock()

	// 启动所有服务器
	if err := a.serverManager.Start(a.ctx); err != nil {
		a.log.Error("服务器启动失败", log.Err(err))
		return err
	}

	// 注册服务
	if a.opts.registrar != nil && a.opts.id != "" {
		serviceInstance := &registry.ServiceInstance{
			ID:        a.opts.id,
			Name:      a.opts.name,
			Version:   a.opts.version,
			Metadata:  a.opts.metadata,
			Status:    registry.StatusUp,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		for _, srv := range a.opts.servers {
			if r, ok := srv.(transport.Endpointer); ok {
				if endpoint, err := r.Endpoint(); err == nil && endpoint != nil {
					serviceInstance.Endpoints = append(serviceInstance.Endpoints, endpoint.String())
				}
			}
		}

		if len(serviceInstance.Endpoints) > 0 {
			if err := a.opts.registrar.Register(a.ctx, serviceInstance); err != nil {
				a.log.Error("服务注册失败", log.Err(err))
			} else {
				a.log.Info("服务注册成功", log.String("id", serviceInstance.ID))
			}
		}
	}

	// 等待信号和处理优雅关闭
	server.WaitForSignal(a.log, func() {
		a.Stop()
	})

	return nil
}

// Stop 停止应用程序
func (a *application) Stop() error {
	a.log.Info("停止应用程序", log.String("id", a.opts.id), log.String("name", a.opts.name))
	a.cancel()

	// 解除服务注册
	if a.opts.registrar != nil && a.opts.id != "" {
		// 创建新的上下文用于注销操作
		ctx, cancel := context.WithTimeout(context.Background(), a.opts.stopTimeout)
		defer cancel()

		if err := a.opts.registrar.Deregister(ctx, &registry.ServiceInstance{ID: a.opts.id}); err != nil {
			a.log.Error("服务注销失败", log.Err(err))
		} else {
			a.log.Info("服务注销成功", log.String("id", a.opts.id))
		}
	}

	// 使用服务器管理器停止所有服务器
	stopCtx, cancel := context.WithTimeout(context.Background(), a.opts.stopTimeout)
	defer cancel()

	if err := a.serverManager.Stop(stopCtx); err != nil {
		a.log.Error("服务器停止失败", log.Err(err))
		return err
	}

	return nil
}
