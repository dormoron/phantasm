package version

import (
	"errors"
	"sync"
	"time"

	"github.com/dormoron/phantasm/config"
)

// Version 是配置版本
type Version struct {
	// ID 是版本ID
	ID string
	// Timestamp 是版本时间戳
	Timestamp time.Time
	// Description 是版本描述
	Description string
	// Data 是版本数据
	Data map[string]interface{}
}

// Manager 是配置版本管理器接口
type Manager interface {
	// Save 保存当前配置为新版本
	Save(description string) (string, error)
	// Rollback 回滚到指定版本
	Rollback(versionID string) error
	// List 列出所有版本
	List() ([]*Version, error)
	// Get 获取指定版本
	Get(versionID string) (*Version, error)
	// Compare 比较两个版本
	Compare(versionID1, versionID2 string) (map[string]interface{}, error)
}

// Option 是版本管理器选项函数
type Option func(*options)

// options 是版本管理器选项
type options struct {
	maxVersions int
}

// WithMaxVersions 设置最大版本数
func WithMaxVersions(max int) Option {
	return func(o *options) {
		o.maxVersions = max
	}
}

// manager 是版本管理器实现
type manager struct {
	opts     options
	config   config.Config
	versions []*Version
	lock     sync.RWMutex
}

// New 创建一个版本管理器
func New(cfg config.Config, opts ...Option) Manager {
	o := options{
		maxVersions: 10, // 默认保留10个版本
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &manager{
		opts:     o,
		config:   cfg,
		versions: []*Version{},
	}
}

// Save 保存当前配置为新版本
func (m *manager) Save(description string) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 获取当前配置数据
	data, err := m.config.Value("").Map()
	if err != nil {
		return "", err
	}

	// 转换为普通map
	dataMap := make(map[string]interface{})
	for k, v := range data {
		// 简化处理，实际应该递归转换
		dataMap[k] = v
	}

	// 创建新版本
	versionID := generateVersionID()
	version := &Version{
		ID:          versionID,
		Timestamp:   time.Now(),
		Description: description,
		Data:        dataMap,
	}

	// 添加到版本列表
	m.versions = append(m.versions, version)

	// 如果超过最大版本数，删除最旧的版本
	if len(m.versions) > m.opts.maxVersions {
		m.versions = m.versions[1:]
	}

	return versionID, nil
}

// Rollback 回滚到指定版本
func (m *manager) Rollback(versionID string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 查找版本
	var targetVersion *Version
	for _, v := range m.versions {
		if v.ID == versionID {
			targetVersion = v
			break
		}
	}

	if targetVersion == nil {
		return errors.New("version not found")
	}

	// 回滚配置
	// 注意：这里只是一个简化的实现，实际应该将版本数据应用到配置中
	// 由于config接口没有提供直接设置值的方法，这里只是一个示例
	return nil
}

// List 列出所有版本
func (m *manager) List() ([]*Version, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// 返回版本列表的副本
	result := make([]*Version, len(m.versions))
	copy(result, m.versions)

	return result, nil
}

// Get 获取指定版本
func (m *manager) Get(versionID string) (*Version, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// 查找版本
	for _, v := range m.versions {
		if v.ID == versionID {
			// 返回版本的副本
			return &Version{
				ID:          v.ID,
				Timestamp:   v.Timestamp,
				Description: v.Description,
				Data:        v.Data,
			}, nil
		}
	}

	return nil, errors.New("version not found")
}

// Compare 比较两个版本
func (m *manager) Compare(versionID1, versionID2 string) (map[string]interface{}, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// 查找版本
	var v1, v2 *Version
	for _, v := range m.versions {
		if v.ID == versionID1 {
			v1 = v
		}
		if v.ID == versionID2 {
			v2 = v
		}
	}

	if v1 == nil || v2 == nil {
		return nil, errors.New("version not found")
	}

	// 比较版本
	diff := make(map[string]interface{})
	for k, v := range v1.Data {
		if v2Val, ok := v2.Data[k]; ok {
			if v != v2Val {
				diff[k] = map[string]interface{}{
					"old": v,
					"new": v2Val,
				}
			}
		} else {
			diff[k] = map[string]interface{}{
				"old": v,
				"new": nil,
			}
		}
	}

	for k, v := range v2.Data {
		if _, ok := v1.Data[k]; !ok {
			diff[k] = map[string]interface{}{
				"old": nil,
				"new": v,
			}
		}
	}

	return diff, nil
}

// generateVersionID 生成版本ID
func generateVersionID() string {
	// 简化实现，使用时间戳作为版本ID
	return time.Now().Format("20060102150405")
}
