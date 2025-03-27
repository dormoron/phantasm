package memory

import (
	"context"
	"sync"
	"time"

	"github.com/dormoron/phantasm/registry"
)

type memRegistry struct {
	sync.RWMutex
	services map[string][]*registry.ServiceInstance
	watchers map[string][]chan []*registry.ServiceInstance
}

// NewRegistry 创建内存注册中心
func NewRegistry() registry.ServiceRegistrar {
	return &memRegistry{
		services: make(map[string][]*registry.ServiceInstance),
		watchers: make(map[string][]chan []*registry.ServiceInstance),
	}
}

// Register 注册服务实例
func (m *memRegistry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	m.Lock()
	defer m.Unlock()

	if service.Status == "" {
		service.Status = registry.StatusUp
	}

	// 设置创建和更新时间
	now := time.Now()
	if service.CreatedAt.IsZero() {
		service.CreatedAt = now
	}
	service.UpdatedAt = now

	// 移除已存在的同ID实例
	services := m.services[service.Name]
	for i, s := range services {
		if s.ID == service.ID {
			services = append(services[:i], services[i+1:]...)
			break
		}
	}

	// 添加服务实例
	m.services[service.Name] = append(services, service)

	// 通知所有监视者
	for _, ch := range m.watchers[service.Name] {
		select {
		case ch <- m.services[service.Name]:
		default:
			// 避免阻塞
		}
	}

	return nil
}

// Deregister 注销服务实例
func (m *memRegistry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	m.Lock()
	defer m.Unlock()

	services := m.services[service.Name]
	for i, s := range services {
		if s.ID == service.ID {
			m.services[service.Name] = append(services[:i], services[i+1:]...)
			// 通知所有监视者
			for _, ch := range m.watchers[service.Name] {
				select {
				case ch <- m.services[service.Name]:
				default:
					// 避免阻塞
				}
			}
			return nil
		}
	}

	return nil
}

// GetService 获取服务实例列表
func (m *memRegistry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	m.RLock()
	defer m.RUnlock()

	services := m.services[serviceName]
	if len(services) == 0 {
		return nil, nil
	}

	// 返回服务列表的副本
	instances := make([]*registry.ServiceInstance, len(services))
	for i, service := range services {
		s := *service
		instances[i] = &s
	}

	return instances, nil
}

// Watch 监视服务变化
func (m *memRegistry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	m.Lock()
	defer m.Unlock()

	// 为服务创建一个新的通知通道
	ch := make(chan []*registry.ServiceInstance, 10)
	m.watchers[serviceName] = append(m.watchers[serviceName], ch)

	// 立即发送当前服务列表
	if services, ok := m.services[serviceName]; ok && len(services) > 0 {
		ch <- services
	}

	return &memWatcher{
		registry: m,
		ch:       ch,
		service:  serviceName,
	}, nil
}

type memWatcher struct {
	registry *memRegistry
	ch       chan []*registry.ServiceInstance
	service  string
	done     bool
}

// Next 等待下一个服务更新
func (w *memWatcher) Next() ([]*registry.ServiceInstance, error) {
	if w.done {
		return nil, context.Canceled
	}

	select {
	case services := <-w.ch:
		return services, nil
	}
}

// Stop 停止监视
func (w *memWatcher) Stop() error {
	w.registry.Lock()
	defer w.registry.Unlock()

	if w.done {
		return nil
	}

	w.done = true
	// 从注册表中移除此监视器
	for i, ch := range w.registry.watchers[w.service] {
		if ch == w.ch {
			w.registry.watchers[w.service] = append(
				w.registry.watchers[w.service][:i],
				w.registry.watchers[w.service][i+1:]...,
			)
			break
		}
	}

	close(w.ch)
	return nil
}
