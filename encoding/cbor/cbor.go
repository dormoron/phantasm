package cbor

import (
	"github.com/fxamacker/cbor/v2"

	"github.com/dormoron/phantasm/encoding"
)

const (
	// Name 是编解码器的名称
	Name = "cbor"
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 是CBOR编解码器的实现
type codec struct{}

// Marshal 将对象序列化为CBOR字节数组
func (codec) Marshal(v interface{}) ([]byte, error) {
	return cbor.Marshal(v)
}

// Unmarshal 将CBOR字节数组反序列化为对象
func (codec) Unmarshal(data []byte, v interface{}) error {
	return cbor.Unmarshal(data, v)
}

// Name 返回编解码器的名称
func (codec) Name() string {
	return Name
}
