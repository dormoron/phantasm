package example

import (
	"fmt"
	"log"

	// 导入所有编解码器
	"github.com/dormoron/phantasm/encoding"
	_ "github.com/dormoron/phantasm/encoding/all"
)

// Address 地址结构
type Address struct {
	Street  string `json:"street" xml:"street" yaml:"street" toml:"street" msgpack:"street" bson:"street"`
	City    string `json:"city" xml:"city" yaml:"city" toml:"city" msgpack:"city" bson:"city"`
	State   string `json:"state" xml:"state" yaml:"state" toml:"state" msgpack:"state" bson:"state"`
	ZipCode string `json:"zip_code" xml:"zip_code" yaml:"zip_code" toml:"zip_code" msgpack:"zip_code" bson:"zip_code"`
}

// Employee 员工结构
type Employee struct {
	ID        int      `json:"id" xml:"id" yaml:"id" toml:"id" msgpack:"id" bson:"id"`
	Name      string   `json:"name" xml:"name" yaml:"name" toml:"name" msgpack:"name" bson:"name"`
	Email     string   `json:"email" xml:"email" yaml:"email" toml:"email" msgpack:"email" bson:"email"`
	Address   Address  `json:"address" xml:"address" yaml:"address" toml:"address" msgpack:"address" bson:"address"`
	Skills    []string `json:"skills" xml:"skills" yaml:"skills" toml:"skills" msgpack:"skills" bson:"skills"`
	IsManager bool     `json:"is_manager" xml:"is_manager" yaml:"is_manager" toml:"is_manager" msgpack:"is_manager" bson:"is_manager"`
}

// RunAdvancedExample 运行高级编码模块示例
func RunAdvancedExample() {
	// 创建示例对象
	employee := &Employee{
		ID:    12345,
		Name:  "李四",
		Email: "lisi@example.com",
		Address: Address{
			Street:  "科技路100号",
			City:    "北京",
			State:   "北京",
			ZipCode: "100000",
		},
		Skills:    []string{"Go", "Rust", "Kubernetes", "Cloud Native"},
		IsManager: true,
	}

	fmt.Println("====== TOML 编解码示例 ======")
	// 获取TOML编解码器
	tomlCodec := encoding.GetCodec("toml")
	if tomlCodec == nil {
		log.Fatal("未找到TOML编解码器")
	}

	// TOML序列化
	tomlData, err := tomlCodec.Marshal(employee)
	if err != nil {
		log.Fatalf("TOML序列化失败: %v", err)
	}
	fmt.Printf("TOML序列化结果:\n%s\n", tomlData)

	// TOML反序列化
	newEmployee := &Employee{}
	if err := tomlCodec.Unmarshal(tomlData, newEmployee); err != nil {
		log.Fatalf("TOML反序列化失败: %v", err)
	}
	fmt.Printf("TOML反序列化结果: %+v\n\n", newEmployee)

	fmt.Println("====== MessagePack 编解码示例 ======")
	// 获取MessagePack编解码器
	msgpackCodec := encoding.GetCodec("msgpack")
	if msgpackCodec == nil {
		log.Fatal("未找到MessagePack编解码器")
	}

	// MessagePack序列化
	msgpackData, err := msgpackCodec.Marshal(employee)
	if err != nil {
		log.Fatalf("MessagePack序列化失败: %v", err)
	}
	fmt.Printf("MessagePack序列化结果长度: %d 字节\n", len(msgpackData))

	// MessagePack反序列化
	newEmployee = &Employee{}
	if err := msgpackCodec.Unmarshal(msgpackData, newEmployee); err != nil {
		log.Fatalf("MessagePack反序列化失败: %v", err)
	}
	fmt.Printf("MessagePack反序列化结果: %+v\n\n", newEmployee)

	fmt.Println("====== CBOR 编解码示例 ======")
	// 获取CBOR编解码器
	cborCodec := encoding.GetCodec("cbor")
	if cborCodec == nil {
		log.Fatal("未找到CBOR编解码器")
	}

	// CBOR序列化
	cborData, err := cborCodec.Marshal(employee)
	if err != nil {
		log.Fatalf("CBOR序列化失败: %v", err)
	}
	fmt.Printf("CBOR序列化结果长度: %d 字节\n", len(cborData))

	// CBOR反序列化
	newEmployee = &Employee{}
	if err := cborCodec.Unmarshal(cborData, newEmployee); err != nil {
		log.Fatalf("CBOR反序列化失败: %v", err)
	}
	fmt.Printf("CBOR反序列化结果: %+v\n\n", newEmployee)

	fmt.Println("====== BSON 编解码示例 ======")
	// 获取BSON编解码器
	bsonCodec := encoding.GetCodec("bson")
	if bsonCodec == nil {
		log.Fatal("未找到BSON编解码器")
	}

	// BSON序列化
	bsonData, err := bsonCodec.Marshal(employee)
	if err != nil {
		log.Fatalf("BSON序列化失败: %v", err)
	}
	fmt.Printf("BSON序列化结果长度: %d 字节\n", len(bsonData))

	// BSON反序列化
	newEmployee = &Employee{}
	if err := bsonCodec.Unmarshal(bsonData, newEmployee); err != nil {
		log.Fatalf("BSON反序列化失败: %v", err)
	}
	fmt.Printf("BSON反序列化结果: %+v\n\n", newEmployee)

	// 比较不同编解码器的压缩率
	fmt.Println("====== 编解码器压缩率比较 ======")
	jsonData, _ := encoding.MarshalJSON(employee)
	yamlData, _ := encoding.MarshalYAML(employee)
	xmlData, _ := encoding.GetCodec("xml").Marshal(employee)

	fmt.Printf("JSON 大小: %d 字节\n", len(jsonData))
	fmt.Printf("YAML 大小: %d 字节\n", len(yamlData))
	fmt.Printf("XML 大小: %d 字节\n", len(xmlData))
	fmt.Printf("TOML 大小: %d 字节\n", len(tomlData))
	fmt.Printf("MessagePack 大小: %d 字节\n", len(msgpackData))
	fmt.Printf("CBOR 大小: %d 字节\n", len(cborData))
	fmt.Printf("BSON 大小: %d 字节\n", len(bsonData))
}
