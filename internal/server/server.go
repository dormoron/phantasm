package server

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dormoron/phantasm/log"
)

// Server 是服务器接口的定义
type Server interface {
	// Start 启动服务器
	Start(context.Context) error
	// Stop 停止服务器
	Stop(context.Context) error
}

// Manager 是服务器管理器的接口
type Manager interface {
	// Add 添加一个服务器
	Add(Server)
	// Start 启动所有服务器
	Start(context.Context) error
	// Stop 停止所有服务器
	Stop(context.Context) error
}

var _ Manager = (*manager)(nil)

// manager 是服务器管理器的具体实现
type manager struct {
	servers []Server
	logger  log.Logger
	lock    sync.Mutex
}

// NewManager 创建一个新的服务器管理器
func NewManager(logger log.Logger) Manager {
	return &manager{
		servers: make([]Server, 0),
		logger:  logger,
	}
}

// Add 向管理器添加一个服务器
func (m *manager) Add(srv Server) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.servers = append(m.servers, srv)
}

// Start 启动所有服务器
func (m *manager) Start(ctx context.Context) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.servers) == 0 {
		return errors.New("没有服务器可启动")
	}

	wg := sync.WaitGroup{}
	done := make(chan error, len(m.servers))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, srv := range m.servers {
		wg.Add(1)
		go func(srv Server) {
			defer wg.Done()
			if err := srv.Start(ctx); err != nil {
				done <- err
				cancel()
			}
		}(srv)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Stop 停止所有服务器
func (m *manager) Stop(ctx context.Context) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.servers) == 0 {
		return nil
	}

	wg := sync.WaitGroup{}
	done := make(chan error, len(m.servers))

	// 使用独立的上下文，避免主上下文取消影响关闭过程
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, srv := range m.servers {
		wg.Add(1)
		go func(srv Server) {
			defer wg.Done()
			if err := srv.Stop(stopCtx); err != nil {
				done <- err
			}
		}(srv)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	var errs []error
	for err := range done {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		// 简单起见，只返回第一个错误
		return errs[0]
	}
	return nil
}

// WaitForSignal 等待系统信号并执行回调
func WaitForSignal(logger log.Logger, callback func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-signals
		logger.Info("收到系统信号", log.String("signal", s.String()))
		if callback != nil {
			callback()
		}
	}()
}

// Address 是服务器地址信息
type Address struct {
	Network string // 网络类型，如 "tcp"
	Address string // 地址，如 ":8080"
	Path    string // 路径，适用于一些特殊服务器
}

// NewAddress 从URL创建地址信息
func NewAddress(endpoint *url.URL) *Address {
	return &Address{
		Network: "tcp",
		Address: endpoint.Host,
		Path:    endpoint.Path,
	}
}

// String 返回地址的字符串表示
func (a *Address) String() string {
	if a.Path == "" {
		return a.Network + "://" + a.Address
	}
	return a.Network + "://" + a.Address + a.Path
}
