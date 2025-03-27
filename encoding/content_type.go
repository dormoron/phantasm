package encoding

import (
	"mime"
	"strings"
)

// 常见的内容类型常量
const (
	// MIME类型
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMEXML2              = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
	MIMEPROTOBUF          = "application/x-protobuf"
	MIMEYAML              = "application/x-yaml"
	MIMEYAML2             = "text/yaml"
	MIMETOML              = "application/toml"
	MIMETOML2             = "text/toml"
	MIMEMSGPACK           = "application/msgpack"
	MIMEMSGPACK2          = "application/x-msgpack"
	MIMECBOR              = "application/cbor"
	MIMECBOR2             = "application/x-cbor"
	MIMEBSON              = "application/bson"
)

// GetCodecForContentType 根据内容类型获取相应的编解码器
func GetCodecForContentType(contentType string) Codec {
	contentType = parseContentType(contentType)

	switch contentType {
	case MIMEJSON:
		return GetCodec("json")
	case MIMEXML, MIMEXML2:
		return GetCodec("xml")
	case MIMEPlain, MIMEPOSTForm:
		return GetCodec("form")
	case MIMEPROTOBUF:
		return GetCodec("proto")
	case MIMEYAML, MIMEYAML2:
		return GetCodec("yaml")
	case MIMETOML, MIMETOML2:
		return GetCodec("toml")
	case MIMEMSGPACK, MIMEMSGPACK2:
		return GetCodec("msgpack")
	case MIMECBOR, MIMECBOR2:
		return GetCodec("cbor")
	case MIMEBSON:
		return GetCodec("bson")
	default:
		// 默认使用JSON
		return GetCodec("json")
	}
}

// parseContentType 解析内容类型，移除参数部分
func parseContentType(contentType string) string {
	if contentType == "" {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.TrimSpace(strings.Split(contentType, ";")[0])
	}

	return mediaType
}
