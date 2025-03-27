package toml

import (
	"github.com/pelletier/go-toml/v2"

	"github.com/dormoron/phantasm/encoding"
)

const (
	// Name 是编解码器的名称
	Name = "toml"
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 是TOML编解码器的实现
type codec struct{}

// Marshal 将对象序列化为TOML字节数组
func (codec) Marshal(v interface{}) ([]byte, error) {
	return toml.Marshal(v)
}

// Unmarshal 将TOML字节数组反序列化为对象
func (codec) Unmarshal(data []byte, v interface{}) error {
	return toml.Unmarshal(data, v)
}

// Name 返回编解码器的名称
func (codec) Name() string {
	return Name
}
