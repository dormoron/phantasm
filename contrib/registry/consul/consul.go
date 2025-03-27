package consul

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/registry"

	"github.com/hashicorp/consul/api"
)

var (
	_ registry.ServiceRegistrar = (*Registry)(nil)
)

// Registry 是基于Consul的服务注册发现中心
type Registry struct {
	client     *api.Client
	ctx        context.Context
	cancel     context.CancelFunc
	lock       sync.RWMutex
	watchChs   map[string]chan []*registry.ServiceInstance // 服务名称到观察通道的映射
	registered map[string]struct{}                         // 已注册的服务实例ID集合
	logger     log.Logger
	options    *Options
}

// Options 是Consul注册中心的选项
type Options struct {
	HealthCheck                    bool          // 是否启用健康检查
	TTL                            time.Duration // 健康检查TTL
	Interval                       time.Duration // 健康检查间隔
	DeregisterCriticalServiceAfter time.Duration // 服务不健康多久后注销
	Logger                         log.Logger
}

// Option 是Consul注册中心的选项函数
type Option func(*Options)

// WithHealthCheck 设置是否启用健康检查
func WithHealthCheck(enable bool) Option {
	return func(o *Options) {
		o.HealthCheck = enable
	}
}

// WithTTL 设置健康检查TTL
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// WithInterval 设置健康检查间隔
func WithInterval(interval time.Duration) Option {
	return func(o *Options) {
		o.Interval = interval
	}
}

// WithDeregisterCriticalServiceAfter 设置服务不健康后自动注销时间
func WithDeregisterCriticalServiceAfter(timeout time.Duration) Option {
	return func(o *Options) {
		o.DeregisterCriticalServiceAfter = timeout
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// NewRegistry 创建Consul注册中心实例
func NewRegistry(client *api.Client, opts ...Option) *Registry {
	options := &Options{
		HealthCheck:                    true,
		TTL:                            time.Second * 15,
		Interval:                       time.Second * 10,
		DeregisterCriticalServiceAfter: time.Minute * 1,
		Logger:                         log.DefaultLogger,
	}

	for _, o := range opts {
		o(options)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Registry{
		client:     client,
		ctx:        ctx,
		cancel:     cancel,
		lock:       sync.RWMutex{},
		watchChs:   make(map[string]chan []*registry.ServiceInstance),
		registered: make(map[string]struct{}),
		logger:     options.Logger,
		options:    options,
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

	var endpoint string
	if len(service.Endpoints) > 0 {
		endpoint = service.Endpoints[0]
	} else {
		return fmt.Errorf("没有可用的服务端点")
	}

	// 解析端点地址
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	// 解析端口
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	// 构建Consul服务定义
	serviceID := service.ID
	reg := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    service.Name,
		Address: host,
		Port:    port,
		Tags:    []string{u.Scheme, service.Version},
		Meta:    service.Metadata,
	}

	// 添加健康检查
	check := &api.AgentServiceCheck{
		DeregisterCriticalServiceAfter: "1m",
	}

	switch u.Scheme {
	case "http":
		check.HTTP = endpoint + "/health"
		check.Interval = "10s"
		check.Timeout = "5s"
	case "grpc":
		check.GRPC = fmt.Sprintf("%s/%s", u.Host, service.Name)
		check.Interval = "10s"
		check.Timeout = "5s"
	default:
		// 默认使用TTL检查
		check.TTL = fmt.Sprintf("%ds", int(r.options.TTL.Seconds()))
	}

	reg.Check = check

	// 注册服务
	if err := r.client.Agent().ServiceRegister(reg); err != nil {
		return err
	}

	// 记录已注册服务
	r.lock.Lock()
	r.registered[serviceID] = struct{}{}
	r.lock.Unlock()

	// 如果是TTL检查，需要启动定期更新健康状态
	if check.TTL != "" {
		go func() {
			ticker := time.NewTicker(r.options.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-r.ctx.Done():
					return
				case <-ticker.C:
					err := r.client.Agent().UpdateTTL("service:"+serviceID, "健康", api.HealthPassing)
					if err != nil {
						r.logger.Error("更新服务TTL失败",
							log.String("service", service.Name),
							log.String("id", serviceID),
							log.String("error", err.Error()))
					}
				}
			}
		}()
	}

	return nil
}

// Deregister 注销服务实例
func (r *Registry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	serviceID := service.ID
	// 注销服务
	if err := r.client.Agent().ServiceDeregister(serviceID); err != nil {
		return err
	}

	// 移除记录
	r.lock.Lock()
	delete(r.registered, serviceID)
	r.lock.Unlock()

	return nil
}

// GetService 获取服务实例列表
func (r *Registry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	entries, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, err
	}

	items := make([]*registry.ServiceInstance, 0, len(entries))
	for _, entry := range entries {
		service := entry.Service
		if service.ID == "" || service.Service == "" {
			continue
		}

		// 构建端点列表
		endpoints := make([]string, 0, len(service.Tags))
		for _, tag := range service.Tags {
			if tag == "http" || tag == "grpc" {
				endpoint := fmt.Sprintf("%s://%s:%d", tag, service.Address, service.Port)
				endpoints = append(endpoints, endpoint)
			}
		}

		if len(endpoints) == 0 {
			// 如果没有显式协议标签，默认使用http
			endpoint := fmt.Sprintf("http://%s:%d", service.Address, service.Port)
			endpoints = append(endpoints, endpoint)
		}

		// 解析版本信息
		var version string
		for _, tag := range service.Tags {
			if tag != "http" && tag != "grpc" {
				version = tag
				break
			}
		}

		// 构建服务实例
		instance := &registry.ServiceInstance{
			ID:        service.ID,
			Name:      service.Service,
			Version:   version,
			Metadata:  service.Meta,
			Endpoints: endpoints,
			Status:    registry.StatusUp,
		}

		items = append(items, instance)
	}

	return items, nil
}

// Watch 监视服务变更
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
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
	go func() {
		index := uint64(0)
		for {
			select {
			case <-r.ctx.Done():
				return
			default:
				// 通过阻塞查询监视服务变化
				entries, meta, err := r.client.Health().Service(serviceName, "", true, &api.QueryOptions{
					WaitIndex: index,
					WaitTime:  time.Minute,
				})
				if err != nil {
					r.logger.Error("监视服务变更失败",
						log.String("service", serviceName),
						log.String("error", err.Error()))
					time.Sleep(time.Second) // 避免频繁重试
					continue
				}

				// 检查索引是否变化
				if meta.LastIndex <= index {
					continue
				}
				index = meta.LastIndex

				// 构建服务实例列表
				items := make([]*registry.ServiceInstance, 0, len(entries))
				for _, entry := range entries {
					service := entry.Service
					if service.ID == "" || service.Service == "" {
						continue
					}

					// 构建端点列表
					endpoints := make([]string, 0, len(service.Tags))
					for _, tag := range service.Tags {
						if tag == "http" || tag == "grpc" {
							endpoint := fmt.Sprintf("%s://%s:%d", tag, service.Address, service.Port)
							endpoints = append(endpoints, endpoint)
						}
					}

					if len(endpoints) == 0 {
						endpoint := fmt.Sprintf("http://%s:%d", service.Address, service.Port)
						endpoints = append(endpoints, endpoint)
					}

					// 解析版本信息
					var version string
					for _, tag := range service.Tags {
						if tag != "http" && tag != "grpc" {
							version = tag
							break
						}
					}

					// 构建服务实例
					instance := &registry.ServiceInstance{
						ID:        service.ID,
						Name:      service.Service,
						Version:   version,
						Metadata:  service.Meta,
						Endpoints: endpoints,
						Status:    registry.StatusUp,
					}

					items = append(items, instance)
				}

				// 通知观察者
				r.lock.RLock()
				ch, ok := r.watchChs[serviceName]
				r.lock.RUnlock()
				if ok {
					select {
					case ch <- items:
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
	return fmt.Sprintf("service/%s/%s", service.Name, service.ID)
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
