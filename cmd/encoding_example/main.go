package main

import (
	"flag"
	"fmt"
	"github.com/dormoron/phantasm/encoding/example"
)

func main() {
	// 解析命令行参数
	mode := flag.String("mode", "basic", "示例模式: basic 或 advanced")
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Println("  Phantasm Encoding 模块示例")
	fmt.Println("==================================================")

	switch *mode {
	case "basic":
		fmt.Println("运行基础编码示例...")
		example.RunExample()
	case "advanced":
		fmt.Println("运行高级编码示例...")
		example.RunAdvancedExample()
	default:
		fmt.Printf("未知模式: %s\n", *mode)
		fmt.Println("请使用 -mode=basic 或 -mode=advanced")
	}
}
