package nacos

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dormoron/phantasm/config"
	"github.com/dormoron/phantasm/log"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"
)

// Format 配置格式类型
type Format string

const (
	// FormatJSON JSON格式
	FormatJSON Format = "json"
	// FormatYAML YAML格式
	FormatYAML Format = "yaml"
	// FormatTOML TOML格式
	FormatTOML Format = "toml"
)

var (
	_ config.Source = (*Source)(nil)
)

// Source Nacos配置源
type Source struct {
	client       config_client.IConfigClient
	options      *options
	watcherGroup sync.Map
	decoders     sync.Map
	data         atomic.Value
	done         chan struct{}
	lock         sync.RWMutex
	logger       log.Logger
}

// options Nacos配置源选项
type options struct {
	ServerConfigs    []constant.ServerConfig
	ClientConfig     constant.ClientConfig
	DataIDs          []string
	Group            string
	NamespaceID      string
	Format           Format
	RefreshInterval  time.Duration
	EnableAutoUpdate bool
	Logger           log.Logger
}

// Option Nacos配置源选项设置函数
type Option func(*options)

// WithServerConfigs 设置Nacos服务器配置
func WithServerConfigs(configs []constant.ServerConfig) Option {
	return func(o *options) {
		o.ServerConfigs = configs
	}
}

// WithServerAddresses 设置Nacos服务器地址
func WithServerAddresses(addresses []string) Option {
	return func(o *options) {
		serverConfigs := make([]constant.ServerConfig, 0, len(addresses))
		for _, addr := range addresses {
			serverConfigs = append(serverConfigs, constant.ServerConfig{
				IpAddr: addr,
				Port:   8848,
			})
		}
		o.ServerConfigs = serverConfigs
	}
}

// WithClientConfig 设置Nacos客户端配置
func WithClientConfig(config constant.ClientConfig) Option {
	return func(o *options) {
		o.ClientConfig = config
	}
}

// WithDataIDs 设置配置数据ID列表
func WithDataIDs(dataIDs []string) Option {
	return func(o *options) {
		o.DataIDs = dataIDs
	}
}

// WithGroup 设置配置分组
func WithGroup(group string) Option {
	return func(o *options) {
		o.Group = group
	}
}

// WithNamespaceID 设置命名空间ID
func WithNamespaceID(namespaceID string) Option {
	return func(o *options) {
		o.NamespaceID = namespaceID
	}
}

// WithFormat 设置配置格式
func WithFormat(format Format) Option {
	return func(o *options) {
		o.Format = format
	}
}

// WithRefreshInterval 设置刷新间隔
func WithRefreshInterval(interval time.Duration) Option {
	return func(o *options) {
		o.RefreshInterval = interval
	}
}

// WithEnableAutoUpdate 设置是否启用自动更新
func WithEnableAutoUpdate(enable bool) Option {
	return func(o *options) {
		o.EnableAutoUpdate = enable
	}
}

// WithLogger 设置日志器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.Logger = logger
	}
}

// NewSource 创建新的Nacos配置源
func NewSource(opts ...Option) (*Source, error) {
	options := &options{
		ServerConfigs: []constant.ServerConfig{
			{
				IpAddr: "127.0.0.1",
				Port:   8848,
			},
		},
		ClientConfig: constant.ClientConfig{
			TimeoutMs:           5000,
			NotLoadCacheAtStart: true,
			LogDir:              "/tmp/nacos/log",
			CacheDir:            "/tmp/nacos/cache",
		},
		Group:            "DEFAULT_GROUP",
		Format:           FormatJSON,
		RefreshInterval:  time.Second * 30,
		EnableAutoUpdate: true,
		Logger:           log.DefaultLogger,
	}

	for _, o := range opts {
		o(options)
	}

	// 设置命名空间
	if options.NamespaceID != "" {
		options.ClientConfig.NamespaceId = options.NamespaceID
	}

	// 创建配置客户端
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ServerConfigs: options.ServerConfigs,
			ClientConfig:  &options.ClientConfig,
		},
	)
	if err != nil {
		return nil, err
	}

	source := &Source{
		client:  client,
		options: options,
		done:    make(chan struct{}),
		logger:  options.Logger,
	}

	// 初始化读取配置
	if err := source.loadAll(); err != nil {
		return nil, err
	}

	// 启动自动刷新
	if options.EnableAutoUpdate {
		// 设置监听
		for _, dataID := range options.DataIDs {
			if err := source.watch(dataID); err != nil {
				source.logger.Error("监听配置失败", log.String("dataID", dataID), log.String("error", err.Error()))
			}
		}
	}

	return source, nil
}

// Load 从nacos加载配置
func (s *Source) Load() ([]*config.KeyValue, error) {
	data, ok := s.data.Load().([]*config.KeyValue)
	if !ok || len(data) == 0 {
		return nil, errors.New("无法加载配置数据")
	}
	return data, nil
}

// Watch 监听配置变更
func (s *Source) Watch() (config.Watcher, error) {
	watcher := &watcher{
		source: s,
		ch:     make(chan []*config.KeyValue, 10),
		done:   make(chan struct{}, 1),
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// 存储watcher
	id := time.Now().UnixNano()
	s.watcherGroup.Store(id, watcher)

	return watcher, nil
}

// loadAll 加载所有配置
func (s *Source) loadAll() error {
	var configs []*config.KeyValue

	for _, dataID := range s.options.DataIDs {
		// 从Nacos获取配置
		content, err := s.client.GetConfig(vo.ConfigParam{
			DataId: dataID,
			Group:  s.options.Group,
		})
		if err != nil {
			s.logger.Error("获取配置失败", log.String("dataID", dataID), log.String("error", err.Error()))
			continue
		}

		// 解析配置
		kv, err := s.parseConfig(dataID, content)
		if err != nil {
			s.logger.Error("解析配置失败", log.String("dataID", dataID), log.String("error", err.Error()))
			continue
		}

		configs = append(configs, kv...)
	}

	// 存储配置
	s.data.Store(configs)

	return nil
}

// watch 监听单个配置
func (s *Source) watch(dataID string) error {
	return s.client.ListenConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  s.options.Group,
		OnChange: func(namespace, group, dataId, data string) {
			s.logger.Info("配置已更新",
				log.String("namespace", namespace),
				log.String("group", group),
				log.String("dataId", dataId))

			// 解析更新的配置
			kvs, err := s.parseConfig(dataId, data)
			if err != nil {
				s.logger.Error("解析更新的配置失败", log.String("dataID", dataId), log.String("error", err.Error()))
				return
			}

			// 更新本地配置
			if err := s.updateConfig(kvs); err != nil {
				s.logger.Error("更新本地配置失败", log.String("dataID", dataId), log.String("error", err.Error()))
				return
			}

			// 通知所有观察者
			s.notifyWatchers()
		},
	})
}

// Decoder 配置解码器接口
type Decoder interface {
	Decode(v interface{}) error
}

// parseConfig 解析配置
func (s *Source) parseConfig(dataID, content string) ([]*config.KeyValue, error) {
	decoder, ok := s.decoders.Load(dataID)
	if !ok {
		// 根据格式创建解码器
		var dec Decoder
		switch s.options.Format {
		case FormatJSON:
			dec = json.NewDecoder(strings.NewReader(content))
		case FormatYAML:
			dec = yaml.NewDecoder(strings.NewReader(content))
		case FormatTOML:
			dec = toml.NewDecoder(strings.NewReader(content))
		default:
			// 默认为原始格式
			return []*config.KeyValue{
				{
					Key:   dataID,
					Value: string([]byte(content)),
				},
			}, nil
		}

		s.decoders.Store(dataID, dec)
		decoder = dec
	}

	var data map[string]interface{}
	if err := decoder.(Decoder).Decode(&data); err != nil {
		return nil, err
	}

	// 将map扁平化为键值对列表
	var kvs []*config.KeyValue
	for k, v := range data {
		val, err := json.Marshal(v)
		if err != nil {
			s.logger.Error("编码配置值失败", log.String("key", k), log.String("error", err.Error()))
			continue
		}

		kvs = append(kvs, &config.KeyValue{
			Key:   k,
			Value: string(val),
		})
	}

	return kvs, nil
}

// updateConfig 更新本地配置
func (s *Source) updateConfig(kvs []*config.KeyValue) error {
	// 获取当前配置
	current, ok := s.data.Load().([]*config.KeyValue)
	if !ok {
		current = make([]*config.KeyValue, 0)
	}

	// 更新配置
	updated := make([]*config.KeyValue, 0, len(current))
	for _, kv := range current {
		// 检查是否在更新列表中
		found := false
		for _, newKV := range kvs {
			if kv.Key == newKV.Key {
				// 使用新值替换旧值
				updated = append(updated, newKV)
				found = true
				break
			}
		}

		// 如果没有找到，保留原值
		if !found {
			updated = append(updated, kv)
		}
	}

	// 添加新的键值对
	for _, newKV := range kvs {
		found := false
		for _, kv := range current {
			if kv.Key == newKV.Key {
				found = true
				break
			}
		}

		if !found {
			updated = append(updated, newKV)
		}
	}

	// 存储更新后的配置
	s.data.Store(updated)

	return nil
}

// notifyWatchers 通知所有观察者
func (s *Source) notifyWatchers() {
	data, ok := s.data.Load().([]*config.KeyValue)
	if !ok || len(data) == 0 {
		return
	}

	// 通知所有watcher
	s.watcherGroup.Range(func(key, value interface{}) bool {
		w, ok := value.(*watcher)
		if !ok {
			return true
		}

		select {
		case w.ch <- data:
		default:
			// 避免阻塞
		}

		return true
	})
}

// Close 关闭配置源
func (s *Source) Close() error {
	close(s.done)
	return nil
}

// watcher 配置观察者实现
type watcher struct {
	source *Source
	ch     chan []*config.KeyValue
	done   chan struct{}
}

// Next 获取下一次配置变更
func (w *watcher) Next() ([]*config.KeyValue, error) {
	select {
	case data := <-w.ch:
		return data, nil
	case <-w.done:
		return nil, errors.New("watcher已关闭")
	}
}

// Stop 停止观察者
func (w *watcher) Stop() error {
	select {
	case <-w.done:
		return nil
	default:
		close(w.done)

		// 从source中移除
		w.source.watcherGroup.Range(func(key, value interface{}) bool {
			if value == w {
				w.source.watcherGroup.Delete(key)
				return false
			}
			return true
		})
	}

	return nil
}
