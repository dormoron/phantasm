package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dormoron/phantasm/log"
	"github.com/dormoron/phantasm/middleware"
)

// Option 是缓存中间件的选项
type Option func(*options)

// WithTTL 设置缓存过期时间
func WithTTL(ttl time.Duration) Option {
	return func(o *options) {
		o.ttl = ttl
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithMaxSize 设置缓存最大条目数
func WithMaxSize(size int) Option {
	return func(o *options) {
		o.maxSize = size
	}
}

// KeyGenerator 定义缓存键生成函数类型
type KeyGenerator func(ctx context.Context, req interface{}) string

// WithKeyGenerator 设置缓存键生成器
func WithKeyGenerator(gen KeyGenerator) Option {
	return func(o *options) {
		o.keyGenerator = gen
	}
}

// CacheStore 缓存存储接口
type CacheStore interface {
	// Get 获取缓存值
	Get(key string) (interface{}, bool)
	// Set 设置缓存值
	Set(key string, value interface{}, ttl time.Duration)
	// Delete 删除缓存值
	Delete(key string)
}

// WithStore 设置缓存存储实现
func WithStore(store CacheStore) Option {
	return func(o *options) {
		o.store = store
	}
}

// options 是缓存中间件的选项
type options struct {
	ttl          time.Duration
	logger       log.Logger
	maxSize      int
	keyGenerator KeyGenerator
	store        CacheStore
}

// inMemoryStore 内存缓存实现
type inMemoryStore struct {
	items    map[string]cacheItem
	mutex    sync.RWMutex
	maxSize  int
	lastUsed []string // 用于LRU淘汰策略
}

// cacheItem 缓存项
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// newInMemoryStore 创建内存缓存存储
func newInMemoryStore(maxSize int) *inMemoryStore {
	return &inMemoryStore{
		items:    make(map[string]cacheItem),
		maxSize:  maxSize,
		lastUsed: make([]string, 0, maxSize),
	}
}

// Get 实现CacheStore.Get
func (s *inMemoryStore) Get(key string) (interface{}, bool) {
	s.mutex.RLock()
	item, found := s.items[key]
	s.mutex.RUnlock()

	if !found {
		return nil, false
	}

	// 检查是否已过期
	if time.Now().After(item.expiration) {
		s.Delete(key)
		return nil, false
	}

	// 更新LRU列表
	s.updateLRU(key)

	return item.value, true
}

// Set 实现CacheStore.Set
func (s *inMemoryStore) Set(key string, value interface{}, ttl time.Duration) {
	expiration := time.Now().Add(ttl)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查是否需要淘汰
	if len(s.items) >= s.maxSize && s.items[key].value == nil {
		s.evict()
	}

	s.items[key] = cacheItem{
		value:      value,
		expiration: expiration,
	}

	// 更新LRU列表
	s.updateLRULocked(key)
}

// Delete 实现CacheStore.Delete
func (s *inMemoryStore) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.items, key)

	// 从LRU列表中删除
	for i, k := range s.lastUsed {
		if k == key {
			s.lastUsed = append(s.lastUsed[:i], s.lastUsed[i+1:]...)
			break
		}
	}
}

// updateLRU 更新LRU列表
func (s *inMemoryStore) updateLRU(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.updateLRULocked(key)
}

// updateLRULocked 更新LRU列表（已加锁）
func (s *inMemoryStore) updateLRULocked(key string) {
	// 从列表中移除
	for i, k := range s.lastUsed {
		if k == key {
			s.lastUsed = append(s.lastUsed[:i], s.lastUsed[i+1:]...)
			break
		}
	}

	// 添加到列表末尾（最近使用）
	s.lastUsed = append(s.lastUsed, key)
}

// evict 淘汰最近最少使用的缓存项
func (s *inMemoryStore) evict() {
	if len(s.lastUsed) == 0 {
		return
	}

	// 淘汰最近最少使用的项
	oldestKey := s.lastUsed[0]
	delete(s.items, oldestKey)
	s.lastUsed = s.lastUsed[1:]
}

// defaultKeyGenerator 默认的缓存键生成器
func defaultKeyGenerator(ctx context.Context, req interface{}) string {
	// 尝试从上下文获取路径和方法
	path, _ := ctx.Value("path").(string)
	method, _ := ctx.Value("method").(string)

	// 序列化请求对象
	var reqData []byte
	if req != nil {
		var err error
		reqData, err = json.Marshal(req)
		if err != nil {
			reqData = []byte(fmt.Sprintf("%v", req))
		}
	}

	// 计算请求体的MD5哈希
	hash := md5.Sum(reqData)
	reqHash := hex.EncodeToString(hash[:])

	// 生成最终的缓存键
	return fmt.Sprintf("%s:%s:%s", method, path, reqHash)
}

// Cache 返回一个缓存中间件
func Cache(opts ...Option) middleware.Middleware {
	options := options{
		ttl:          time.Minute * 5,
		logger:       log.DefaultLogger,
		maxSize:      1000,
		keyGenerator: defaultKeyGenerator,
	}
	for _, o := range opts {
		o(&options)
	}

	// 如果未提供存储，使用内存存储
	if options.store == nil {
		options.store = newInMemoryStore(options.maxSize)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 生成缓存键
			key := options.keyGenerator(ctx, req)

			// 检查缓存
			if cached, found := options.store.Get(key); found {
				options.logger.Debug("Cache hit",
					log.String("key", key),
					log.String("path", fmt.Sprintf("%v", ctx.Value("path"))),
				)
				return cached, nil
			}

			// 缓存未命中，执行处理程序
			options.logger.Debug("Cache miss",
				log.String("key", key),
				log.String("path", fmt.Sprintf("%v", ctx.Value("path"))),
			)

			resp, err := handler(ctx, req)

			// 如果处理成功，缓存响应
			if err == nil && resp != nil {
				options.store.Set(key, resp, options.ttl)
			}

			return resp, err
		}
	}
}
