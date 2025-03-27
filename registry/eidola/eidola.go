package eidola

import (
	"context"
	"encoding/json"

	"github.com/dormoron/eidola/registry"

	phantasm_registry "github.com/dormoron/phantasm/registry"
)

// Adapter 适配器用于将eidola的registry适配为github.com/dormoron/phantasm的registry
type Adapter struct {
	eidolaRegistry registry.Registry
}

// NewAdapter 创建一个新的适配器
func NewAdapter(eidolaRegistry registry.Registry) *Adapter {
	return &Adapter{
		eidolaRegistry: eidolaRegistry,
	}
}

// Register 注册服务实例
func (a *Adapter) Register(ctx context.Context, service *phantasm_registry.ServiceInstance) error {
	// 将github.com/dormoron/phantasm的ServiceInstance转换为eidola的ServiceInstance
	eservice := registry.ServiceInstance{
		Name:    service.Name,
		Address: getFirstEndpoint(service.Endpoints),
		Weight:  1,               // 默认权重
		Group:   service.Version, // 使用版本作为分组
	}

	return a.eidolaRegistry.Register(ctx, eservice)
}

// Deregister 注销服务实例
func (a *Adapter) Deregister(ctx context.Context, service *phantasm_registry.ServiceInstance) error {
	// 将github.com/dormoron/phantasm的ServiceInstance转换为eidola的ServiceInstance
	eservice := registry.ServiceInstance{
		Name:    service.Name,
		Address: getFirstEndpoint(service.Endpoints),
	}

	return a.eidolaRegistry.UnRegister(ctx, eservice)
}

// getFirstEndpoint 获取第一个端点地址
func getFirstEndpoint(endpoints []string) string {
	if len(endpoints) > 0 {
		return endpoints[0]
	}
	return ""
}

// EidolaRegistryFactory 创建基于eidola的服务注册器
func EidolaRegistryFactory(eidolaRegistry registry.Registry) (phantasm_registry.Registrar, error) {
	// 创建适配器并返回
	return NewAdapter(eidolaRegistry), nil
}

// StoreMetadata 将元数据存储到服务实例中
// 由于eidola的ServiceInstance结构没有直接的元数据字段
// 我们可以将元数据序列化后存储在Group字段中
func StoreMetadata(metadata map[string]string) (string, error) {
	if len(metadata) == 0 {
		return "", nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
