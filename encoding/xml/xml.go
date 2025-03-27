package xml

import (
	"encoding/xml"

	"github.com/dormoron/phantasm/encoding"
)

const (
	// Name 是编解码器的名称
	Name = "xml"
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 是XML编解码器的实现
type codec struct{}

// Marshal 将对象序列化为XML字节数组
func (codec) Marshal(v interface{}) ([]byte, error) {
	return xml.Marshal(v)
}

// Unmarshal 将XML字节数组反序列化为对象
func (codec) Unmarshal(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}

// Name 返回编解码器的名称
func (codec) Name() string {
	return Name
}
