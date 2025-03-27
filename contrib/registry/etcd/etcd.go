package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/registry"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	_ registry.ServiceRegistrar = (*Registry)(nil)
)

// Registry 是基于etcd的服务注册发现中心
type Registry struct {
	client   *clientv3.Client
	prefix   string // 服务注册的前缀
	ttl      int64
	ctx      context.Context
	cancel   context.CancelFunc
	lock     sync.RWMutex
	leases   map[string]clientv3.LeaseID                 // 服务实例ID到租约ID的映射
	watchChs map[string]chan []*registry.ServiceInstance // 服务名称到观察通道的映射
	logger   log.Logger
}

// Options 是etcd注册中心的选项
type Options struct {
	Prefix string        // 注册前缀
	TTL    time.Duration // 租约有效期
	Logger log.Logger
}

// Option 是etcd注册中心的选项函数
type Option func(*Options)

// WithPrefix 设置注册前缀
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithTTL 设置租约有效期
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// NewRegistry 创建etcd注册中心实例
func NewRegistry(client *clientv3.Client, opts ...Option) *Registry {
	options := &Options{
		Prefix: "/services",
		TTL:    time.Second * 15,
		Logger: log.DefaultLogger,
	}

	for _, o := range opts {
		o(options)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Registry{
		client:   client,
		prefix:   options.Prefix,
		ttl:      int64(options.TTL.Seconds()),
		ctx:      ctx,
		cancel:   cancel,
		lock:     sync.RWMutex{},
		leases:   make(map[string]clientv3.LeaseID),
		watchChs: make(map[string]chan []*registry.ServiceInstance),
		logger:   options.Logger,
	}
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

	val, err := json.Marshal(service)
	if err != nil {
		return err
	}

	key := r.serviceKey(service)

	// 创建租约
	lease, err := r.client.Grant(ctx, r.ttl)
	if err != nil {
		return err
	}

	// 使用租约注册服务
	_, err = r.client.Put(ctx, key, string(val), clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}

	// 保持租约有效
	ch, err := r.client.KeepAlive(r.ctx, lease.ID)
	if err != nil {
		return err
	}

	// 启动goroutine接收keepalive响应
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				return
			case _, ok := <-ch:
				if !ok {
					r.logger.Warn("续租失败，尝试重新注册", log.String("service", service.Name), log.String("id", service.ID))
					// 尝试重新注册
					if err := r.Register(context.Background(), service); err != nil {
						r.logger.Error("服务重新注册失败", log.String("service", service.Name), log.String("id", service.ID), log.Err(err))
					}
					return
				}
			}
		}
	}()

	// 保存租约信息
	r.lock.Lock()
	r.leases[service.ID] = lease.ID
	r.lock.Unlock()

	return nil
}

// Deregister 注销服务实例
func (r *Registry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	key := r.serviceKey(service)

	// 获取租约并撤销
	r.lock.Lock()
	leaseID, ok := r.leases[service.ID]
	if ok {
		delete(r.leases, service.ID)
	}
	r.lock.Unlock()

	if ok {
		_, err := r.client.Revoke(ctx, leaseID)
		if err != nil {
			return err
		}
	}

	// 删除服务实例
	_, err := r.client.Delete(ctx, key)
	return err
}

// GetService 获取服务实例列表
func (r *Registry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	prefix := path.Join(r.prefix, serviceName) + "/"
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	items := make([]*registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		si := &registry.ServiceInstance{}
		if err := json.Unmarshal(kv.Value, si); err != nil {
			r.logger.Warn("解析服务实例失败", log.String("key", string(kv.Key)), log.Err(err))
			continue
		}
		if si.Status != registry.StatusDown {
			items = append(items, si)
		}
	}

	return items, nil
}

// Watch 监视服务变更
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	prefix := path.Join(r.prefix, serviceName) + "/"

	// 创建一个新的通道
	r.lock.Lock()
	ch := make(chan []*registry.ServiceInstance, 10)
	r.watchChs[serviceName] = ch
	r.lock.Unlock()

	// 先获取当前服务列表
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
	watchCh := r.client.Watch(r.ctx, prefix, clientv3.WithPrefix())
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				return
			case resp, ok := <-watchCh:
				if !ok {
					return
				}
				if err := resp.Err(); err != nil {
					r.logger.Error("监视错误", log.String("service", serviceName), log.Err(err))
					continue
				}
				if len(resp.Events) == 0 {
					continue
				}

				// 获取最新的服务列表
				services, err := r.GetService(context.Background(), serviceName)
				if err != nil {
					r.logger.Error("获取服务列表失败", log.String("service", serviceName), log.Err(err))
					continue
				}

				// 通知所有观察者
				r.lock.RLock()
				ch, ok := r.watchChs[serviceName]
				r.lock.RUnlock()
				if ok {
					select {
					case ch <- services:
					default:
						// 避免阻塞
					}
				}
			}
		}
	}()

	return &watcher{
		registry: r,
		ch:       ch,
		service:  serviceName,
	}, nil
}

// Stop 停止注册中心
func (r *Registry) Stop() error {
	r.cancel()
	return nil
}

// serviceKey 构建服务实例的key
func (r *Registry) serviceKey(service *registry.ServiceInstance) string {
	return path.Join(r.prefix, service.Name, service.ID)
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
