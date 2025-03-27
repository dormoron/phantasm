package validate

import (
	"errors"
	"reflect"
	"strings"

	"github.com/dormoron/phantasm/config"
)

// Validator 是配置验证器接口
type Validator interface {
	// Validate 验证配置值
	Validate(value config.Value) error
}

// Rule 是验证规则接口
type Rule interface {
	// Validate 验证值
	Validate(value interface{}) error
}

// Option 是验证器选项函数
type Option func(*options)

// options 是验证器选项
type options struct {
	rules map[string][]Rule
}

// WithRule 添加验证规则
func WithRule(key string, rule Rule) Option {
	return func(o *options) {
		if o.rules == nil {
			o.rules = make(map[string][]Rule)
		}
		o.rules[key] = append(o.rules[key], rule)
	}
}

// validator 是验证器实现
type validator struct {
	opts options
}

// New 创建一个验证器
func New(opts ...Option) Validator {
	o := options{
		rules: make(map[string][]Rule),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &validator{opts: o}
}

// Validate 验证配置值
func (v *validator) Validate(value config.Value) error {
	if value == nil {
		return errors.New("value is nil")
	}

	m, err := value.Map()
	if err != nil {
		return err
	}

	for key, rules := range v.opts.rules {
		parts := strings.Split(key, ".")
		val, ok := getNestedValue(m, parts)
		if !ok {
			return errors.New("key not found: " + key)
		}

		for _, rule := range rules {
			if err := rule.Validate(val); err != nil {
				return errors.New(key + ": " + err.Error())
			}
		}
	}

	return nil
}

// getNestedValue 获取嵌套值
func getNestedValue(m map[string]config.Value, parts []string) (interface{}, bool) {
	if len(parts) == 0 {
		return nil, false
	}

	if len(parts) == 1 {
		val, ok := m[parts[0]]
		if !ok {
			return nil, false
		}
		return val, true
	}

	val, ok := m[parts[0]]
	if !ok {
		return nil, false
	}

	nextMap, err := val.Map()
	if err != nil {
		return nil, false
	}

	return getNestedValue(nextMap, parts[1:])
}

// 内置验证规则

// Required 必填规则
type Required struct{}

// Validate 验证值是否为空
func (r Required) Validate(value interface{}) error {
	if value == nil {
		return errors.New("value is required")
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return errors.New("value is required")
		}
	case reflect.Slice, reflect.Map:
		if v.Len() == 0 {
			return errors.New("value is required")
		}
	}

	return nil
}

// Range 范围规则
type Range struct {
	Min, Max float64
}

// Validate 验证值是否在范围内
func (r Range) Validate(value interface{}) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := float64(v.Int())
		if val < r.Min || val > r.Max {
			return errors.New("value out of range")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := float64(v.Uint())
		if val < r.Min || val > r.Max {
			return errors.New("value out of range")
		}
	case reflect.Float32, reflect.Float64:
		val := v.Float()
		if val < r.Min || val > r.Max {
			return errors.New("value out of range")
		}
	default:
		return errors.New("value type not supported")
	}

	return nil
}

// Length 长度规则
type Length struct {
	Min, Max int
}

// Validate 验证值的长度是否在范围内
func (l Length) Validate(value interface{}) error {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		len := len(v.String())
		if len < l.Min || len > l.Max {
			return errors.New("length out of range")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		len := v.Len()
		if len < l.Min || len > l.Max {
			return errors.New("length out of range")
		}
	default:
		return errors.New("value type not supported")
	}

	return nil
}

// Pattern 模式规则
type Pattern struct {
	Regex string
}

// Validate 验证值是否匹配正则表达式
func (p Pattern) Validate(value interface{}) error {
	// 简化实现，实际应该使用正则表达式库
	return errors.New("not implemented")
}
