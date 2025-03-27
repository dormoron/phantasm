package all

import (
	// 导入所有编解码器以便初始化
	_ "github.com/dormoron/phantasm/encoding/bson"
	_ "github.com/dormoron/phantasm/encoding/cbor"
	_ "github.com/dormoron/phantasm/encoding/form"
	_ "github.com/dormoron/phantasm/encoding/json"
	_ "github.com/dormoron/phantasm/encoding/msgpack"
	_ "github.com/dormoron/phantasm/encoding/proto"
	_ "github.com/dormoron/phantasm/encoding/toml"
	_ "github.com/dormoron/phantasm/encoding/xml"
	_ "github.com/dormoron/phantasm/encoding/yaml"
)
