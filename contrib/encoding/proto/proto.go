package proto

import (
	"errors"
	"reflect"

	"github.com/dormoron/phantasm/encoding"

	"google.golang.org/protobuf/proto"
)

// Name 编码器名称
const Name = "proto"

var (
	// ErrNotProtoMessage 不是proto消息
	ErrNotProtoMessage = errors.New("不是有效的proto消息")

	// ErrUnsupportedType 不支持的类型
	ErrUnsupportedType = errors.New("不支持的类型")
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 实现Proto编码器
type codec struct{}

// Marshal 将数据序列化为Proto格式
func (codec) Marshal(v interface{}) ([]byte, error) {
	switch m := v.(type) {
	case proto.Message:
		return proto.Marshal(m)
	case nil:
		return nil, ErrNotProtoMessage
	default:
		// 尝试转换为proto.Message
		if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr {
			if _, ok := rv.Interface().(proto.Message); ok {
				return proto.Marshal(rv.Interface().(proto.Message))
			}
		}
		return nil, ErrNotProtoMessage
	}
}

// Unmarshal 从Proto格式反序列化
func (codec) Unmarshal(data []byte, v interface{}) error {
	switch m := v.(type) {
	case proto.Message:
		return proto.Unmarshal(data, m)
	case nil:
		return ErrNotProtoMessage
	default:
		// 尝试转换为proto.Message
		if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr {
			if msg, ok := rv.Interface().(proto.Message); ok {
				return proto.Unmarshal(data, msg)
			}
		}
		return ErrNotProtoMessage
	}
}

// Name 返回编码器名称
func (codec) Name() string {
	return Name
}
