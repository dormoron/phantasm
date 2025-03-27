package zookeeper

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/registry"

	"github.com/go-zookeeper/zk"
)

var (
	_ registry.ServiceRegistrar = (*Registry)(nil)
)

// Registry 是基于ZooKeeper的服务注册发现中心
type Registry struct {
	conn     *zk.Conn
	prefix   string // 服务注册的前缀
	ctx      context.Context
	cancel   context.CancelFunc
	lock     sync.RWMutex
	watchChs map[string]chan []*registry.ServiceInstance // 服务名称到观察通道的映射
	services map[string]*registry.ServiceInstance        // 本地缓存的服务实例
	logger   log.Logger
}

// Options 是ZooKeeper注册中心的选项
type Options struct {
	Prefix         string        // 注册前缀
	SessionTimeout time.Duration // 会话超时时间
	Logger         log.Logger
}

// Option 是ZooKeeper注册中心的选项函数
type Option func(*Options)

// WithPrefix 设置注册前缀
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithSessionTimeout 设置会话超时时间
func WithSessionTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.SessionTimeout = timeout
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// NewRegistry 创建ZooKeeper注册中心实例
func NewRegistry(servers []string, opts ...Option) (*Registry, error) {
	options := &Options{
		Prefix:         "/services",
		SessionTimeout: time.Second * 15,
		Logger:         log.DefaultLogger,
	}

	for _, o := range opts {
		o(options)
	}

	conn, _, err := zk.Connect(servers, options.SessionTimeout)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Registry{
		conn:     conn,
		prefix:   options.Prefix,
		ctx:      ctx,
		cancel:   cancel,
		lock:     sync.RWMutex{},
		watchChs: make(map[string]chan []*registry.ServiceInstance),
		services: make(map[string]*registry.ServiceInstance),
		logger:   options.Logger,
	}

	// 确保前缀路径存在
	if err := r.ensurePathExists(options.Prefix); err != nil {
		cancel()
		conn.Close()
		return nil, err
	}

	return r, nil
}

// Register 注册服务实例
func (r *Registry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	if service.Status == "" {
		service.Status = registry.StatusUp
	}

	// 设置创建和更新时间
	now := time.Now()
	if service.CreatedAt.IsZero() {
		service.CreatedAt = now
	}
	service.UpdatedAt = now

	// 序列化服务实例
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	// 创建服务路径
	servicePath := path.Join(r.prefix, service.Name)
	if err := r.ensurePathExists(servicePath); err != nil {
		return err
	}

	// 创建服务实例节点
	instancePath := path.Join(servicePath, service.ID)
	exists, stat, err := r.conn.Exists(instancePath)
	if err != nil {
		return err
	}

	if exists {
		// 更新已存在的节点
		_, err = r.conn.Set(instancePath, data, stat.Version)
	} else {
		// 创建新节点，临时节点，当会话关闭时自动删除
		_, err = r.conn.Create(instancePath, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}

	if err != nil {
		return err
	}

	// 缓存服务实例
	r.lock.Lock()
	r.services[service.ID] = service
	r.lock.Unlock()

	return nil
}

// Deregister 注销服务实例
func (r *Registry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	instancePath := path.Join(r.prefix, service.Name, service.ID)
	exists, stat, err := r.conn.Exists(instancePath)
	if err != nil {
		return err
	}

	if exists {
		err = r.conn.Delete(instancePath, stat.Version)
		if err != nil {
			return err
		}
	}

	// 移除缓存的服务实例
	r.lock.Lock()
	delete(r.services, service.ID)
	r.lock.Unlock()

	return nil
}

// GetService 获取服务实例列表
func (r *Registry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	servicePath := path.Join(r.prefix, serviceName)
	exists, _, err := r.conn.Exists(servicePath)
	if err != nil {
		return nil, err
	}

	if !exists {
		// 服务不存在
		return nil, nil
	}

	// 获取所有服务实例ID
	instanceIDs, _, err := r.conn.Children(servicePath)
	if err != nil {
		return nil, err
	}

	instances := make([]*registry.ServiceInstance, 0, len(instanceIDs))
	for _, id := range instanceIDs {
		instancePath := path.Join(servicePath, id)
		data, _, err := r.conn.Get(instancePath)
		if err != nil {
			r.logger.Warn("获取服务实例数据失败",
				log.String("path", instancePath),
				log.String("error", err.Error()))
			continue
		}

		instance := &registry.ServiceInstance{}
		if err := json.Unmarshal(data, instance); err != nil {
			r.logger.Warn("解析服务实例数据失败",
				log.String("path", instancePath),
				log.String("error", err.Error()))
			continue
		}

		if instance.Status != registry.StatusDown {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// Watch 监视服务变更
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	servicePath := path.Join(r.prefix, serviceName)
	exists, _, err := r.conn.Exists(servicePath)
	if err != nil {
		return nil, err
	}

	if !exists {
		// 创建服务路径
		if err := r.ensurePathExists(servicePath); err != nil {
			return nil, err
		}
	}

	// 创建通道
	r.lock.Lock()
	ch := make(chan []*registry.ServiceInstance, 10)
	r.watchChs[serviceName] = ch
	r.lock.Unlock()

	// 获取初始服务列表
	services, err := r.GetService(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	if len(services) > 0 {
		select {
		case ch <- services:
		default:
			// 避免阻塞
		}
	}

	// 启动goroutine监视变更
	go r.watchService(serviceName, servicePath, ch)

	return &watcher{
		registry: r,
		ch:       ch,
		service:  serviceName,
	}, nil
}

// 监视服务变更
func (r *Registry) watchService(serviceName, servicePath string, ch chan []*registry.ServiceInstance) {
	for {
		// 监视子节点变更
		children, _, childEventCh, err := r.conn.ChildrenW(servicePath)
		if err != nil {
			r.logger.Error("监视服务变更失败",
				log.String("service", serviceName),
				log.String("error", err.Error()))

			// 检查节点是否存在，可能需要重新创建
			exists, _, err := r.conn.Exists(servicePath)
			if err != nil || !exists {
				if err := r.ensurePathExists(servicePath); err != nil {
					r.logger.Error("创建服务路径失败",
						log.String("path", servicePath),
						log.String("error", err.Error()))
				}
			}

			// 短暂休眠后重试
			select {
			case <-r.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		// 构建实例列表
		instances := make([]*registry.ServiceInstance, 0, len(children))
		for _, id := range children {
			instancePath := path.Join(servicePath, id)
			data, _, err := r.conn.Get(instancePath)
			if err != nil {
				continue
			}

			instance := &registry.ServiceInstance{}
			if err := json.Unmarshal(data, instance); err != nil {
				continue
			}

			if instance.Status != registry.StatusDown {
				instances = append(instances, instance)
			}
		}

		// 通知观察者
		r.lock.RLock()
		ch, ok := r.watchChs[serviceName]
		r.lock.RUnlock()
		if ok {
			select {
			case ch <- instances:
			default:
				// 避免阻塞
			}
		}

		// 等待事件或上下文取消
		select {
		case <-r.ctx.Done():
			return
		case <-childEventCh:
			// 子节点变更，重新获取服务列表
		}
	}
}

// Stop 停止注册中心
func (r *Registry) Stop() error {
	r.cancel()
	r.conn.Close()
	return nil
}

// 确保路径存在
func (r *Registry) ensurePathExists(path string) error {
	if path == "" {
		return nil
	}

	// 按层次创建路径
	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		current = current + "/" + part
		exists, _, err := r.conn.Exists(current)
		if err != nil {
			return err
		}

		if !exists {
			_, err = r.conn.Create(current, []byte{}, 0, zk.WorldACL(zk.PermAll))
			if err != nil && err != zk.ErrNodeExists {
				return err
			}
		}
	}

	return nil
}

// watcher 是服务监视实现
type watcher struct {
	registry *Registry
	ch       chan []*registry.ServiceInstance
	service  string
	done     bool
}

// Next 等待下一个服务更新
func (w *watcher) Next() ([]*registry.ServiceInstance, error) {
	if w.done {
		return nil, fmt.Errorf("watcher已关闭")
	}

	select {
	case services := <-w.ch:
		return services, nil
	case <-w.registry.ctx.Done():
		return nil, w.registry.ctx.Err()
	}
}

// Stop 停止监视
func (w *watcher) Stop() error {
	if w.done {
		return nil
	}

	w.registry.lock.Lock()
	delete(w.registry.watchChs, w.service)
	w.registry.lock.Unlock()

	w.done = true
	close(w.ch)
	return nil
}
