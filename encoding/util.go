package encoding

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"reflect"
)

// MarshalToString 将对象序列化为字符串
func MarshalToString(codec Codec, v interface{}) (string, error) {
	data, err := codec.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalFromString 从字符串反序列化对象
func UnmarshalFromString(codec Codec, data string, v interface{}) error {
	return codec.Unmarshal([]byte(data), v)
}

// MarshalJSON 将对象序列化为JSON字节数组
func MarshalJSON(v interface{}) ([]byte, error) {
	codec := GetCodec("json")
	if codec == nil {
		return nil, errors.New("JSON编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalJSON 将JSON字节数组反序列化为对象
func UnmarshalJSON(data []byte, v interface{}) error {
	codec := GetCodec("json")
	if codec == nil {
		return errors.New("JSON编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}

// MarshalProto 将对象序列化为Proto字节数组
func MarshalProto(v interface{}) ([]byte, error) {
	codec := GetCodec("proto")
	if codec == nil {
		return nil, errors.New("Proto编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalProto 将Proto字节数组反序列化为对象
func UnmarshalProto(data []byte, v interface{}) error {
	codec := GetCodec("proto")
	if codec == nil {
		return errors.New("Proto编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}

// MarshalYAML 将对象序列化为YAML字节数组
func MarshalYAML(v interface{}) ([]byte, error) {
	codec := GetCodec("yaml")
	if codec == nil {
		return nil, errors.New("YAML编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalYAML 将YAML字节数组反序列化为对象
func UnmarshalYAML(data []byte, v interface{}) error {
	codec := GetCodec("yaml")
	if codec == nil {
		return errors.New("YAML编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}

// DeepCopy 使用gob编码进行深拷贝
func DeepCopy(src, dst interface{}) error {
	if reflect.TypeOf(src) != reflect.TypeOf(dst) {
		return errors.New("源和目标类型不匹配")
	}

	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	dec := gob.NewDecoder(buffer)

	if err := enc.Encode(src); err != nil {
		return err
	}

	return dec.Decode(dst)
}

// CopyReader 将reader内容复制到多个writers
func CopyReader(reader io.Reader, writers ...io.Writer) error {
	if len(writers) == 0 {
		return nil
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	for _, writer := range writers {
		if _, err := writer.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// MarshalToml 将对象序列化为TOML字节数组
func MarshalToml(v interface{}) ([]byte, error) {
	codec := GetCodec("toml")
	if codec == nil {
		return nil, errors.New("TOML编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalToml 将TOML字节数组反序列化为对象
func UnmarshalToml(data []byte, v interface{}) error {
	codec := GetCodec("toml")
	if codec == nil {
		return errors.New("TOML编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}

// MarshalMsgPack 将对象序列化为MessagePack字节数组
func MarshalMsgPack(v interface{}) ([]byte, error) {
	codec := GetCodec("msgpack")
	if codec == nil {
		return nil, errors.New("MessagePack编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalMsgPack 将MessagePack字节数组反序列化为对象
func UnmarshalMsgPack(data []byte, v interface{}) error {
	codec := GetCodec("msgpack")
	if codec == nil {
		return errors.New("MessagePack编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}

// MarshalCbor 将对象序列化为CBOR字节数组
func MarshalCbor(v interface{}) ([]byte, error) {
	codec := GetCodec("cbor")
	if codec == nil {
		return nil, errors.New("CBOR编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalCbor 将CBOR字节数组反序列化为对象
func UnmarshalCbor(data []byte, v interface{}) error {
	codec := GetCodec("cbor")
	if codec == nil {
		return errors.New("CBOR编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}

// MarshalBson 将对象序列化为BSON字节数组
func MarshalBson(v interface{}) ([]byte, error) {
	codec := GetCodec("bson")
	if codec == nil {
		return nil, errors.New("BSON编解码器未注册")
	}
	return codec.Marshal(v)
}

// UnmarshalBson 将BSON字节数组反序列化为对象
func UnmarshalBson(data []byte, v interface{}) error {
	codec := GetCodec("bson")
	if codec == nil {
		return errors.New("BSON编解码器未注册")
	}
	return codec.Unmarshal(data, v)
}
