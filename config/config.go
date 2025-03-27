package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/magiconair/properties"
	"gopkg.in/yaml.v3"
)

var (
	// ErrNotFound 是在键不存在时返回的错误
	ErrNotFound = errors.New("key not found")
	// ErrTypeMismatch 是在值类型与请求不匹配时返回的错误
	ErrTypeMismatch = errors.New("type mismatch")
	// ErrWatcherClosed 是在监视器关闭时返回的错误
	ErrWatcherClosed = errors.New("watcher closed")
)

// Observer 是配置更改通知的观察者
type Observer func(string, Value)

// Value 是配置值接口
type Value interface {
	Bool() (bool, error)
	Int() (int64, error)
	Float() (float64, error)
	String() (string, error)
	Duration() (time.Duration, error)
	Slice() ([]Value, error)
	Map() (map[string]Value, error)
	Scan(interface{}) error
}

// Source 是配置源接口
type Source interface {
	Load() ([]*KeyValue, error)
	Watch() (Watcher, error)
}

// Watcher 是配置更改的观察者
type Watcher interface {
	Next() ([]*KeyValue, error)
	Stop() error
}

// KeyValue 是配置键值对
type KeyValue struct {
	Key   string
	Value string
	// Format 是配置内容的格式，例如 "json", "yaml", "toml", "properties"
	Format string
}

// Config 是配置接口
type Config interface {
	Load() error
	Scan(v interface{}) error
	Value(key string) Value
	Watch(key string, o Observer) error
	Close() error
}

// ValueCallback 是配置值更改的回调
type ValueCallback func(string, Value)

// config 是配置实现
type config struct {
	opts      options
	reader    Reader
	cached    sync.Map
	observers sync.Map
	watchers  []Watcher
}

// New 创建一个配置
func New(opts ...Option) Config {
	o := options{
		sources:  nil,
		decoder:  defaultDecoder,
		resolver: defaultResolver,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &config{
		opts:   o,
		reader: newReader(o),
	}
}

// Load 加载配置源
func (c *config) Load() error {
	for _, src := range c.opts.sources {
		kvs, err := src.Load()
		if err != nil {
			return err
		}
		for _, v := range kvs {
			c.reader.Merge(v)
		}
	}
	return nil
}

// Scan 扫描配置到结构体
func (c *config) Scan(v interface{}) error {
	data, err := c.reader.Values("")
	if err != nil {
		return err
	}
	return c.opts.decoder(data, v)
}

// Value 获取配置值
func (c *config) Value(key string) Value {
	if v, ok := c.cached.Load(key); ok {
		return v.(Value)
	}
	v, err := c.reader.Value(key)
	if err != nil {
		return defaultValue{}
	}
	c.cached.Store(key, v)
	return v
}

// Watch 观察配置更改
func (c *config) Watch(key string, o Observer) error {
	if v, ok := c.observers.Load(key); ok {
		callbacks := v.([]Observer)
		callbacks = append(callbacks, o)
		c.observers.Store(key, callbacks)
	} else {
		c.observers.Store(key, []Observer{o})
	}
	return nil
}

// Close 关闭配置
func (c *config) Close() error {
	for _, w := range c.watchers {
		if err := w.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// defaultValue 是默认值
type defaultValue struct{}

// Bool 返回布尔值
func (v defaultValue) Bool() (bool, error) {
	return false, ErrNotFound
}

// Int 返回整数值
func (v defaultValue) Int() (int64, error) {
	return 0, ErrNotFound
}

// Float 返回浮点值
func (v defaultValue) Float() (float64, error) {
	return 0.0, ErrNotFound
}

// String 返回字符串值
func (v defaultValue) String() (string, error) {
	return "", ErrNotFound
}

// Duration 返回时间间隔
func (v defaultValue) Duration() (time.Duration, error) {
	return 0, ErrNotFound
}

// Slice 返回切片
func (v defaultValue) Slice() ([]Value, error) {
	return nil, ErrNotFound
}

// Map 返回映射
func (v defaultValue) Map() (map[string]Value, error) {
	return nil, ErrNotFound
}

// Scan 扫描到结构体
func (v defaultValue) Scan(interface{}) error {
	return ErrNotFound
}

// Reader 是配置读取器
type Reader interface {
	Merge(*KeyValue)
	Value(string) (Value, error)
	Values(string) (map[string]interface{}, error)
}

// newReader 创建一个新读取器
func newReader(o options) Reader {
	return &reader{
		opts:   o,
		values: make(map[string]interface{}),
	}
}

// reader 是读取器实现
type reader struct {
	opts   options
	values map[string]interface{}
	lock   sync.Mutex
}

// Merge 合并键值
func (r *reader) Merge(kv *KeyValue) {
	r.lock.Lock()
	defer r.lock.Unlock()

	var values map[string]interface{}
	var err error

	switch kv.Format {
	case "json":
		values, err = r.decodeJSON(kv.Value)
	case "yaml":
		values, err = r.decodeYAML(kv.Value)
	case "toml":
		values, err = r.decodeTOML(kv.Value)
	case "properties":
		values, err = r.decodeProperties(kv.Value)
	default:
		values = map[string]interface{}{
			kv.Key: kv.Value,
		}
	}

	if err != nil {
		return
	}

	r.mergeValues(r.values, values)
}

// Value 获取值
func (r *reader) Value(path string) (Value, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	value, ok := r.getNestedValue(r.values, path)
	if !ok {
		return nil, ErrNotFound
	}

	return r.convertToValue(value)
}

// Values 获取值映射
func (r *reader) Values(path string) (map[string]interface{}, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if path == "" {
		return r.values, nil
	}

	value, ok := r.getNestedValue(r.values, path)
	if !ok {
		return nil, ErrNotFound
	}

	if m, ok := value.(map[string]interface{}); ok {
		return m, nil
	}

	return nil, ErrTypeMismatch
}

// defaultDecoder 是默认解码器
func defaultDecoder(src map[string]interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// defaultResolver 是默认解析器
func defaultResolver(str string) string {
	// 支持环境变量解析
	if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") {
		env := strings.TrimSuffix(strings.TrimPrefix(str, "${"), "}")
		return os.Getenv(env)
	}
	return str
}

// 新增的辅助方法

// decodeJSON 解码JSON
func (r *reader) decodeJSON(data string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	return result, err
}

// decodeYAML 解码YAML
func (r *reader) decodeYAML(data string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := yaml.Unmarshal([]byte(data), &result)
	return result, err
}

// decodeTOML 解码TOML
func (r *reader) decodeTOML(data string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := toml.Unmarshal([]byte(data), &result)
	return result, err
}

// decodeProperties 解码Properties
func (r *reader) decodeProperties(data string) (map[string]interface{}, error) {
	props := properties.NewProperties()
	err := props.Load([]byte(data), properties.UTF8)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, key := range props.Keys() {
		value, _ := props.Get(key)
		result[key] = value
	}
	return result, nil
}

// mergeValues 合并值
func (r *reader) mergeValues(dst, src map[string]interface{}) {
	for k, v := range src {
		if m, ok := v.(map[string]interface{}); ok {
			if existing, ok := dst[k].(map[string]interface{}); ok {
				r.mergeValues(existing, m)
			} else {
				dst[k] = m
			}
		} else {
			dst[k] = v
		}
	}
}

// getNestedValue 获取嵌套值
func (r *reader) getNestedValue(values map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := values

	for i, part := range parts {
		if i == len(parts)-1 {
			value, ok := current[part]
			return value, ok
		}

		if m, ok := current[part].(map[string]interface{}); ok {
			current = m
		} else {
			return nil, false
		}
	}

	return nil, false
}

// convertToValue 转换为Value接口
func (r *reader) convertToValue(v interface{}) (Value, error) {
	switch val := v.(type) {
	case bool:
		return boolValue(val), nil
	case int:
		return intValue(val), nil
	case int64:
		return intValue(val), nil
	case float64:
		return floatValue(val), nil
	case string:
		return stringValue(val), nil
	case []interface{}:
		return sliceValue(val), nil
	case map[string]interface{}:
		return mapValue(val), nil
	default:
		return nil, ErrTypeMismatch
	}
}

// 新增的值类型实现
type boolValue bool
type intValue int64
type floatValue float64
type stringValue string
type sliceValue []interface{}
type mapValue map[string]interface{}

func (v boolValue) Bool() (bool, error)              { return bool(v), nil }
func (v boolValue) Int() (int64, error)              { return 0, ErrTypeMismatch }
func (v boolValue) Float() (float64, error)          { return 0, ErrTypeMismatch }
func (v boolValue) String() (string, error)          { return "", ErrTypeMismatch }
func (v boolValue) Duration() (time.Duration, error) { return 0, ErrTypeMismatch }
func (v boolValue) Slice() ([]Value, error)          { return nil, ErrTypeMismatch }
func (v boolValue) Map() (map[string]Value, error)   { return nil, ErrTypeMismatch }
func (v boolValue) Scan(interface{}) error           { return ErrTypeMismatch }

func (v intValue) Bool() (bool, error)              { return false, ErrTypeMismatch }
func (v intValue) Int() (int64, error)              { return int64(v), nil }
func (v intValue) Float() (float64, error)          { return float64(v), nil }
func (v intValue) String() (string, error)          { return "", ErrTypeMismatch }
func (v intValue) Duration() (time.Duration, error) { return time.Duration(v), nil }
func (v intValue) Slice() ([]Value, error)          { return nil, ErrTypeMismatch }
func (v intValue) Map() (map[string]Value, error)   { return nil, ErrTypeMismatch }
func (v intValue) Scan(interface{}) error           { return ErrTypeMismatch }

func (v floatValue) Bool() (bool, error)              { return false, ErrTypeMismatch }
func (v floatValue) Int() (int64, error)              { return int64(v), nil }
func (v floatValue) Float() (float64, error)          { return float64(v), nil }
func (v floatValue) String() (string, error)          { return "", ErrTypeMismatch }
func (v floatValue) Duration() (time.Duration, error) { return 0, ErrTypeMismatch }
func (v floatValue) Slice() ([]Value, error)          { return nil, ErrTypeMismatch }
func (v floatValue) Map() (map[string]Value, error)   { return nil, ErrTypeMismatch }
func (v floatValue) Scan(interface{}) error           { return ErrTypeMismatch }

func (v stringValue) Bool() (bool, error)              { return strconv.ParseBool(string(v)) }
func (v stringValue) Int() (int64, error)              { return strconv.ParseInt(string(v), 10, 64) }
func (v stringValue) Float() (float64, error)          { return strconv.ParseFloat(string(v), 64) }
func (v stringValue) String() (string, error)          { return string(v), nil }
func (v stringValue) Duration() (time.Duration, error) { return time.ParseDuration(string(v)) }
func (v stringValue) Slice() ([]Value, error)          { return nil, ErrTypeMismatch }
func (v stringValue) Map() (map[string]Value, error)   { return nil, ErrTypeMismatch }
func (v stringValue) Scan(interface{}) error           { return ErrTypeMismatch }

func (v sliceValue) Bool() (bool, error)              { return false, ErrTypeMismatch }
func (v sliceValue) Int() (int64, error)              { return 0, ErrTypeMismatch }
func (v sliceValue) Float() (float64, error)          { return 0, ErrTypeMismatch }
func (v sliceValue) String() (string, error)          { return "", ErrTypeMismatch }
func (v sliceValue) Duration() (time.Duration, error) { return 0, ErrTypeMismatch }
func (v sliceValue) Slice() ([]Value, error) {
	values := make([]Value, len(v))
	for i, item := range v {
		switch val := item.(type) {
		case bool:
			values[i] = boolValue(val)
		case int:
			values[i] = intValue(val)
		case int64:
			values[i] = intValue(val)
		case float64:
			values[i] = floatValue(val)
		case string:
			values[i] = stringValue(val)
		case []interface{}:
			values[i] = sliceValue(val)
		case map[string]interface{}:
			values[i] = mapValue(val)
		default:
			return nil, ErrTypeMismatch
		}
	}
	return values, nil
}
func (v sliceValue) Map() (map[string]Value, error) { return nil, ErrTypeMismatch }
func (v sliceValue) Scan(interface{}) error         { return ErrTypeMismatch }

func (v mapValue) Bool() (bool, error)              { return false, ErrTypeMismatch }
func (v mapValue) Int() (int64, error)              { return 0, ErrTypeMismatch }
func (v mapValue) Float() (float64, error)          { return 0, ErrTypeMismatch }
func (v mapValue) String() (string, error)          { return "", ErrTypeMismatch }
func (v mapValue) Duration() (time.Duration, error) { return 0, ErrTypeMismatch }
func (v mapValue) Slice() ([]Value, error)          { return nil, ErrTypeMismatch }
func (v mapValue) Map() (map[string]Value, error) {
	values := make(map[string]Value)
	for k, item := range v {
		switch val := item.(type) {
		case bool:
			values[k] = boolValue(val)
		case int:
			values[k] = intValue(val)
		case int64:
			values[k] = intValue(val)
		case float64:
			values[k] = floatValue(val)
		case string:
			values[k] = stringValue(val)
		case []interface{}:
			values[k] = sliceValue(val)
		case map[string]interface{}:
			values[k] = mapValue(val)
		default:
			return nil, ErrTypeMismatch
		}
	}
	return values, nil
}
func (v mapValue) Scan(interface{}) error { return ErrTypeMismatch }
