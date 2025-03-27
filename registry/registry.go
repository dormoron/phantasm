package registry

import (
	"context"
	"net/url"
	"time"
)

// ServiceInstance 表示一个服务实例
type ServiceInstance struct {
	// ID 是服务实例的唯一标识符
	ID string
	// Name 是服务实例的名称
	Name string
	// Version 是服务实例的版本
	Version string
	// Metadata 是服务实例的元数据
	Metadata map[string]string
	// Endpoints 是服务实例的终端点列表
	Endpoints []string
	// Status 是服务实例的状态
	Status ServiceInstanceStatus
	// CreatedAt 是服务实例的创建时间
	CreatedAt time.Time
	// UpdatedAt 是服务实例的更新时间
	UpdatedAt time.Time
}

// ServiceInstanceStatus 表示服务实例的状态
type ServiceInstanceStatus string

const (
	// StatusUp 表示服务实例正常运行
	StatusUp ServiceInstanceStatus = "UP"
	// StatusDown 表示服务实例已关闭
	StatusDown ServiceInstanceStatus = "DOWN"
	// StatusUnknown 表示服务实例状态未知
	StatusUnknown ServiceInstanceStatus = "UNKNOWN"
)

// Registry 是服务注册接口
type Registry interface {
	// Register 注册服务实例
	Register(ctx context.Context, service *ServiceInstance) error
	// Deregister 注销服务实例
	Deregister(ctx context.Context, service *ServiceInstance) error
}

// Registrar 是服务注册接口，与Registry保持一致
// 这样可以兼容eidola中的registry实现
type Registrar interface {
	// Register 注册服务实例
	Register(ctx context.Context, service *ServiceInstance) error
	// Deregister 注销服务实例
	Deregister(ctx context.Context, service *ServiceInstance) error
}

// Discovery 是服务发现接口
type Discovery interface {
	// GetService 获取服务实例
	GetService(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	// Watch 监视服务更改
	Watch(ctx context.Context, serviceName string) (Watcher, error)
}

// Watcher 是服务更改的观察者
type Watcher interface {
	// Next 返回服务的下一个变更
	Next() ([]*ServiceInstance, error)
	// Stop 停止观察
	Stop() error
}

// RegistryType 表示注册中心类型
type RegistryType string

const (
	// Consul 注册中心类型
	Consul RegistryType = "consul"
	// Etcd 注册中心类型
	Etcd RegistryType = "etcd"
	// Zookeeper 注册中心类型
	Zookeeper RegistryType = "zookeeper"
	// Nacos 注册中心类型
	Nacos RegistryType = "nacos"
	// Eureka 注册中心类型
	Eureka RegistryType = "eureka"
	// Memory 内存注册中心类型
	Memory RegistryType = "memory"
)

// RegistryConfig 是注册中心配置
type RegistryConfig struct {
	// Type 是注册中心类型
	Type RegistryType
	// Endpoints 是注册中心终端点列表
	Endpoints []string
	// Timeout 是注册中心操作超时时间
	Timeout time.Duration
	// Username 是注册中心用户名
	Username string
	// Password 是注册中心密码
	Password string
	// TLSEnabled 是否启用TLS
	TLSEnabled bool
}

// RegistryOption 是注册中心选项函数
type RegistryOption func(o *RegistryConfig)

// WithEndpoints 设置注册中心终端点
func WithEndpoints(endpoints ...string) RegistryOption {
	return func(o *RegistryConfig) {
		o.Endpoints = endpoints
	}
}

// WithTimeout 设置注册中心操作超时时间
func WithTimeout(timeout time.Duration) RegistryOption {
	return func(o *RegistryConfig) {
		o.Timeout = timeout
	}
}

// WithCredentials 设置注册中心认证信息
func WithCredentials(username, password string) RegistryOption {
	return func(o *RegistryConfig) {
		o.Username = username
		o.Password = password
	}
}

// WithTLS 设置是否启用TLS
func WithTLS(enabled bool) RegistryOption {
	return func(o *RegistryConfig) {
		o.TLSEnabled = enabled
	}
}

// ValidateEndpoints 验证终端点列表
func ValidateEndpoints(endpoints []string) error {
	for _, endpoint := range endpoints {
		if _, err := url.Parse(endpoint); err != nil {
			return err
		}
	}
	return nil
}

// ServiceRegistrar 是服务注册器，同时包含注册与发现功能
type ServiceRegistrar interface {
	Registry
	Discovery
}
