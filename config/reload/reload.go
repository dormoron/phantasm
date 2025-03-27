package reload

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dormoron/phantasm/config"
)

// Reloader 是配置热重载器接口
type Reloader interface {
	// Start 启动热重载
	Start() error
	// Stop 停止热重载
	Stop() error
	// SetInterval 设置重载间隔
	SetInterval(interval time.Duration)
	// AddCallback 添加重载回调
	AddCallback(callback func(config.Config) error) error
}

// Option 是热重载器选项函数
type Option func(*options)

// options 是热重载器选项
type options struct {
	interval  time.Duration
	callbacks []func(config.Config) error
}

// WithInterval 设置重载间隔
func WithInterval(interval time.Duration) Option {
	return func(o *options) {
		o.interval = interval
	}
}

// WithCallback 添加重载回调
func WithCallback(callback func(config.Config) error) Option {
	return func(o *options) {
		o.callbacks = append(o.callbacks, callback)
	}
}

// reloader 是热重载器实现
type reloader struct {
	opts      options
	config    config.Config
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	lock      sync.RWMutex
	callbacks []func(config.Config) error
}

// New 创建一个热重载器
func New(cfg config.Config, opts ...Option) Reloader {
	ctx, cancel := context.WithCancel(context.Background())
	o := options{
		interval:  time.Second * 30, // 默认30秒重载一次
		callbacks: []func(config.Config) error{},
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &reloader{
		opts:      o,
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		callbacks: o.callbacks,
	}
}

// Start 启动热重载
func (r *reloader) Start() error {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(r.opts.interval)
		defer ticker.Stop()

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				// 重新加载配置
				if err := r.config.Load(); err != nil {
					// 加载失败，继续使用旧配置
					continue
				}

				// 执行回调
				r.lock.RLock()
				callbacks := r.callbacks
				r.lock.RUnlock()

				for _, callback := range callbacks {
					_ = callback(r.config)
				}
			}
		}
	}()

	return nil
}

// Stop 停止热重载
func (r *reloader) Stop() error {
	r.cancel()
	r.wg.Wait()
	return nil
}

// SetInterval 设置重载间隔
func (r *reloader) SetInterval(interval time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.opts.interval = interval
}

// AddCallback 添加重载回调
func (r *reloader) AddCallback(callback func(config.Config) error) error {
	if callback == nil {
		return errors.New("callback is nil")
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	r.callbacks = append(r.callbacks, callback)
	return nil
}
