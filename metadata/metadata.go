package metadata

import (
	"context"
	"strings"
)

// Metadata 是元数据类型
type Metadata map[string][]string

// New 从键值对创建元数据
func New(m map[string]string) Metadata {
	md := make(Metadata, len(m))
	for k, v := range m {
		key := strings.ToLower(k)
		md[key] = append(md[key], v)
	}
	return md
}

// Get 获取与key关联的第一个值
func (m Metadata) Get(key string) string {
	if vs := m[strings.ToLower(key)]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// Set 设置与key关联的值
func (m Metadata) Set(key, value string) {
	key = strings.ToLower(key)
	m[key] = []string{value}
}

// Add 添加与key关联的值
func (m Metadata) Add(key, value string) {
	key = strings.ToLower(key)
	m[key] = append(m[key], value)
}

// Values 获取与key关联的所有值
func (m Metadata) Values(key string) []string {
	return m[strings.ToLower(key)]
}

// Range 遍历元数据
func (m Metadata) Range(f func(key, value string) bool) {
	for k, vs := range m {
		for _, v := range vs {
			if !f(k, v) {
				return
			}
		}
	}
}

// Clone 克隆元数据
func (m Metadata) Clone() Metadata {
	md := make(Metadata, len(m))
	for k, v := range m {
		values := make([]string, len(v))
		copy(values, v)
		md[k] = values
	}
	return md
}

type metadataKey struct{}

// NewContext 创建带有元数据的新上下文
func NewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, md)
}

// FromContext 从上下文中获取元数据
func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metadataKey{}).(Metadata)
	return md, ok
}

// MergeContext 合并元数据到上下文
func MergeContext(ctx context.Context, md Metadata) context.Context {
	// 尝试获取现有的元数据
	existingMD, ok := FromContext(ctx)
	if !ok {
		return NewContext(ctx, md)
	}
	// 复制和合并现有元数据
	merged := existingMD.Clone()
	for k, vs := range md {
		for _, v := range vs {
			merged.Add(k, v)
		}
	}
	return NewContext(ctx, merged)
}
