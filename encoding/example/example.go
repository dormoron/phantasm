package example

import (
	"fmt"
	"log"

	// 导入所有编解码器
	"github.com/dormoron/phantasm/encoding"
	_ "github.com/dormoron/phantasm/encoding/all"
)

// Person 示例结构体
type Person struct {
	Name    string   `json:"name" xml:"name" yaml:"name" form:"name"`
	Age     int      `json:"age" xml:"age" yaml:"age" form:"age"`
	Email   string   `json:"email" xml:"email" yaml:"email" form:"email"`
	Hobbies []string `json:"hobbies" xml:"hobbies" yaml:"hobbies" form:"hobbies"`
}

// RunExample 运行编码模块示例
func RunExample() {
	// 创建示例对象
	person := &Person{
		Name:    "张三",
		Age:     28,
		Email:   "zhangsan@example.com",
		Hobbies: []string{"阅读", "旅行", "编程"},
	}

	// 获取JSON编解码器
	jsonCodec := encoding.GetCodec("json")
	if jsonCodec == nil {
		log.Fatal("未找到JSON编解码器")
	}

	// JSON序列化
	jsonData, err := jsonCodec.Marshal(person)
	if err != nil {
		log.Fatalf("JSON序列化失败: %v", err)
	}
	fmt.Printf("JSON序列化结果: %s\n", jsonData)

	// JSON反序列化
	newPerson := &Person{}
	if err := jsonCodec.Unmarshal(jsonData, newPerson); err != nil {
		log.Fatalf("JSON反序列化失败: %v", err)
	}
	fmt.Printf("JSON反序列化结果: %+v\n", newPerson)

	// 获取XML编解码器
	xmlCodec := encoding.GetCodec("xml")
	if xmlCodec == nil {
		log.Fatal("未找到XML编解码器")
	}

	// XML序列化
	xmlData, err := xmlCodec.Marshal(person)
	if err != nil {
		log.Fatalf("XML序列化失败: %v", err)
	}
	fmt.Printf("XML序列化结果: %s\n", xmlData)

	// XML反序列化
	newPerson = &Person{}
	if err := xmlCodec.Unmarshal(xmlData, newPerson); err != nil {
		log.Fatalf("XML反序列化失败: %v", err)
	}
	fmt.Printf("XML反序列化结果: %+v\n", newPerson)

	// 获取YAML编解码器
	yamlCodec := encoding.GetCodec("yaml")
	if yamlCodec == nil {
		log.Fatal("未找到YAML编解码器")
	}

	// YAML序列化
	yamlData, err := yamlCodec.Marshal(person)
	if err != nil {
		log.Fatalf("YAML序列化失败: %v", err)
	}
	fmt.Printf("YAML序列化结果:\n%s\n", yamlData)

	// YAML反序列化
	newPerson = &Person{}
	if err := yamlCodec.Unmarshal(yamlData, newPerson); err != nil {
		log.Fatalf("YAML反序列化失败: %v", err)
	}
	fmt.Printf("YAML反序列化结果: %+v\n", newPerson)

	// 获取Form编解码器
	formCodec := encoding.GetCodec("form")
	if formCodec == nil {
		log.Fatal("未找到Form编解码器")
	}

	// Form序列化
	formData, err := formCodec.Marshal(person)
	if err != nil {
		log.Fatalf("Form序列化失败: %v", err)
	}
	fmt.Printf("Form序列化结果: %s\n", formData)

	// Form反序列化
	newPerson = &Person{}
	if err := formCodec.Unmarshal(formData, newPerson); err != nil {
		log.Fatalf("Form反序列化失败: %v", err)
	}
	fmt.Printf("Form反序列化结果: %+v\n", newPerson)

	// 使用辅助函数
	jsonStr, err := encoding.MarshalToString(jsonCodec, person)
	if err != nil {
		log.Fatalf("转换为JSON字符串失败: %v", err)
	}
	fmt.Printf("JSON字符串: %s\n", jsonStr)

	// 根据内容类型获取编解码器
	contentType := "application/json; charset=utf-8"
	codec := encoding.GetCodecForContentType(contentType)
	fmt.Printf("内容类型 %s 对应的编解码器: %s\n", contentType, codec.Name())

	// 深拷贝
	copiedPerson := &Person{}
	if err := encoding.DeepCopy(person, copiedPerson); err != nil {
		log.Fatalf("深拷贝失败: %v", err)
	}
	fmt.Printf("深拷贝结果: %+v\n", copiedPerson)
}
