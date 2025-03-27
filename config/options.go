package config

// Option 是配置选项函数
type Option func(*options)

// options 是配置选项
type options struct {
	sources  []Source
	decoder  Decoder
	resolver Resolver
}

// Decoder 是配置解码器函数类型
type Decoder func(src map[string]interface{}, dst interface{}) error

// Resolver 是配置解析器函数类型
type Resolver func(string) string

// WithSource 添加配置源
func WithSource(s ...Source) Option {
	return func(o *options) {
		o.sources = append(o.sources, s...)
	}
}

// WithDecoder 设置解码器
func WithDecoder(d Decoder) Option {
	return func(o *options) {
		o.decoder = d
	}
}

// WithResolver 设置解析器
func WithResolver(r Resolver) Option {
	return func(o *options) {
		o.resolver = r
	}
}
