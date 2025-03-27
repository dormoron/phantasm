package proto

import (
	"errors"

	"google.golang.org/protobuf/proto"

	"github.com/dormoron/phantasm/encoding"
)

const (
	// Name 是编解码器的名称
	Name = "proto"
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 是Proto编解码器的实现
type codec struct{}

// Marshal 将对象序列化为Proto字节数组
func (codec) Marshal(v interface{}) ([]byte, error) {
	message, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("传入的对象不是proto.Message类型")
	}
	return proto.Marshal(message)
}

// Unmarshal 将Proto字节数组反序列化为对象
func (codec) Unmarshal(data []byte, v interface{}) error {
	message, ok := v.(proto.Message)
	if !ok {
		return errors.New("传入的对象不是proto.Message类型")
	}
	return proto.Unmarshal(data, message)
}

// Name 返回编解码器的名称
func (codec) Name() string {
	return Name
}
