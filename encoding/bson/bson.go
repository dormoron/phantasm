package bson

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/dormoron/phantasm/encoding"
)

const (
	// Name 是编解码器的名称
	Name = "bson"
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 是BSON编解码器的实现
type codec struct{}

// Marshal 将对象序列化为BSON字节数组
func (codec) Marshal(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

// Unmarshal 将BSON字节数组反序列化为对象
func (codec) Unmarshal(data []byte, v interface{}) error {
	return bson.Unmarshal(data, v)
}

// Name 返回编解码器的名称
func (codec) Name() string {
	return Name
}
