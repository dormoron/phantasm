package encoding

import (
	"reflect"
)

// Codec 定义了编解码器的接口
type Codec interface {
	// Marshal 将对象序列化为字节数组
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal 将字节数组反序列化为对象
	Unmarshal(data []byte, v interface{}) error
	// Name 返回编解码器的名称
	Name() string
}

var (
	// codecs 存储所有已注册的编解码器
	codecs = make(map[string]Codec)
)

// RegisterCodec 注册一个编解码器
func RegisterCodec(codec Codec) {
	if codec == nil {
		panic("不能注册空的编解码器")
	}
	name := codec.Name()
	if name == "" {
		panic("编解码器名称不能为空")
	}
	if _, ok := codecs[name]; ok {
		panic("编解码器 " + name + " 已经注册")
	}
	codecs[name] = codec
}

// GetCodec 根据名称获取编解码器
func GetCodec(name string) Codec {
	return codecs[name]
}

// Clone 克隆一个对象
func Clone(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	original := reflect.ValueOf(src)
	if original.Kind() == reflect.Ptr {
		original = original.Elem()
	}

	cpy := reflect.New(original.Type())
	copyRecursive(original, cpy.Elem())
	return cpy.Interface()
}

// copyRecursive 递归复制对象的值
func copyRecursive(src, dst reflect.Value) {
	if src.CanInterface() {
		if !dst.CanSet() {
			return
		}

		switch src.Kind() {
		case reflect.Ptr:
			if src.IsNil() {
				return
			}
			dst.Set(reflect.New(src.Elem().Type()))
			copyRecursive(src.Elem(), dst.Elem())
		case reflect.Interface:
			if src.IsNil() {
				return
			}
			srcElem := src.Elem()
			dstElem := reflect.New(srcElem.Type()).Elem()
			copyRecursive(srcElem, dstElem)
			dst.Set(dstElem)
		case reflect.Struct:
			for i := 0; i < src.NumField(); i++ {
				if dst.Field(i).CanSet() {
					copyRecursive(src.Field(i), dst.Field(i))
				}
			}
		case reflect.Slice:
			if src.IsNil() {
				return
			}
			dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Cap()))
			for i := 0; i < src.Len(); i++ {
				copyRecursive(src.Index(i), dst.Index(i))
			}
		case reflect.Map:
			if src.IsNil() {
				return
			}
			dst.Set(reflect.MakeMap(src.Type()))
			for _, key := range src.MapKeys() {
				dstVal := reflect.New(src.MapIndex(key).Type()).Elem()
				copyRecursive(src.MapIndex(key), dstVal)
				dst.SetMapIndex(key, dstVal)
			}
		default:
			dst.Set(src)
		}
	}
}
