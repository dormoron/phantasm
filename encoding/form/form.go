package form

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/dormoron/phantasm/encoding"
)

const (
	// Name 是编解码器的名称
	Name = "form"
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec 是Form编解码器的实现
type codec struct{}

// Marshal 将对象序列化为Form字节数组
func (codec) Marshal(v interface{}) ([]byte, error) {
	values, err := encodeToValues(v)
	if err != nil {
		return nil, err
	}
	return []byte(values.Encode()), nil
}

// Unmarshal 将Form字节数组反序列化为对象
func (codec) Unmarshal(data []byte, v interface{}) error {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	return decodeValues(values, v)
}

// Name 返回编解码器的名称
func (codec) Name() string {
	return Name
}

// encodeToValues 将对象编码为url.Values
func encodeToValues(v interface{}) (url.Values, error) {
	values := make(url.Values)
	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return values, nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, errors.New("无法将非结构体编码为form")
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Anonymous {
			continue
		}

		tag := field.Tag.Get("form")
		if tag == "" {
			tag = field.Tag.Get("json") // 尝试使用json标签
		}
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		if tag == "-" {
			continue
		}

		name := parseTag(tag)
		if name == "" {
			continue
		}

		fv := rv.Field(i)
		if !fv.CanInterface() {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			values.Set(name, fv.String())
		case reflect.Bool:
			values.Set(name, strconv.FormatBool(fv.Bool()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			values.Set(name, strconv.FormatInt(fv.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			values.Set(name, strconv.FormatUint(fv.Uint(), 10))
		case reflect.Float32, reflect.Float64:
			values.Set(name, strconv.FormatFloat(fv.Float(), 'f', -1, 64))
		case reflect.Slice:
			if fv.Type().Elem().Kind() == reflect.String {
				for j := 0; j < fv.Len(); j++ {
					values.Add(name, fv.Index(j).String())
				}
			}
		}
	}

	return values, nil
}

// decodeValues 从url.Values解码到对象
func decodeValues(values url.Values, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("解码目标必须是非nil的指针")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("解码目标必须是结构体指针")
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Anonymous {
			continue
		}

		tag := field.Tag.Get("form")
		if tag == "" {
			tag = field.Tag.Get("json") // 尝试使用json标签
		}
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		if tag == "-" {
			continue
		}

		name := parseTag(tag)
		if name == "" {
			continue
		}

		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}

		vs := values[name]
		if len(vs) == 0 {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(vs[0])
		case reflect.Bool:
			b, err := strconv.ParseBool(vs[0])
			if err != nil {
				return fmt.Errorf("字段 %s: %w", name, err)
			}
			fv.SetBool(b)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(vs[0], 10, 64)
			if err != nil {
				return fmt.Errorf("字段 %s: %w", name, err)
			}
			fv.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, err := strconv.ParseUint(vs[0], 10, 64)
			if err != nil {
				return fmt.Errorf("字段 %s: %w", name, err)
			}
			fv.SetUint(u)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(vs[0], 64)
			if err != nil {
				return fmt.Errorf("字段 %s: %w", name, err)
			}
			fv.SetFloat(f)
		case reflect.Slice:
			if fv.Type().Elem().Kind() == reflect.String {
				sv := reflect.MakeSlice(fv.Type(), len(vs), len(vs))
				for j, v := range vs {
					sv.Index(j).SetString(v)
				}
				fv.Set(sv)
			}
		}
	}

	return nil
}

// parseTag 解析标签获取字段名
func parseTag(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}
