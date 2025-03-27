package proto

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CmdProto 表示生成proto的命令
var CmdProto = &cobra.Command{
	Use:   "proto",
	Short: "生成proto代码",
	Long:  `生成proto代码，包括Go、HTTP、gRPC和错误代码`,
	Run:   generateProto,
}

var (
	protoPath string
	outputDir string
)

func init() {
	CmdProto.Flags().StringVarP(&protoPath, "proto-path", "p", "./api", "proto文件的路径")
	CmdProto.Flags().StringVarP(&outputDir, "output", "o", "./api", "生成的代码输出目录")
}

func generateProto(cmd *cobra.Command, args []string) {
	// 检查protoc是否安装
	if err := exec.Command("protoc", "--version").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "未找到protoc，请先安装protoc")
		fmt.Fprintln(os.Stderr, "参考: https://grpc.io/docs/protoc-installation/")
		os.Exit(1)
	}

	// 检查protoc-gen-go和protoc-gen-go-grpc是否安装
	checkProtocPlugin("protoc-gen-go")
	checkProtocPlugin("protoc-gen-go-grpc")

	// 安装phantasm自己的插件
	installPhantasmPlugins()

	// 查找所有proto文件
	var protoFiles []string
	err := filepath.Walk(protoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "查找proto文件时出错: %v\n", err)
		os.Exit(1)
	}

	if len(protoFiles) == 0 {
		fmt.Fprintf(os.Stderr, "在 %s 中没有找到 .proto 文件\n", protoPath)
		os.Exit(1)
	}

	// 为每个proto文件生成代码
	for _, protoFile := range protoFiles {
		fmt.Printf("生成 %s 的代码\n", protoFile)

		// 构建protoc命令
		args := []string{
			"--proto_path=" + filepath.Dir(protoFile),
			"--proto_path=" + protoPath,
			"--go_out=" + outputDir,
			"--go_opt=paths=source_relative",
			"--go-grpc_out=" + outputDir,
			"--go-grpc_opt=paths=source_relative",
			"--go-http_out=" + outputDir,
			"--go-http_opt=paths=source_relative",
			"--go-errors_out=" + outputDir,
			"--go-errors_opt=paths=source_relative",
			protoFile,
		}

		cmd := exec.Command("protoc", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "为 %s 生成代码时出错: %v\n", protoFile, err)
			os.Exit(1)
		}
	}

	fmt.Println("所有proto文件已成功生成代码")
}

func checkProtocPlugin(plugin string) {
	path, err := exec.LookPath(plugin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "未找到 %s，正在安装...\n", plugin)
		cmd := exec.Command("go", "install", "google.golang.org/protobuf/cmd/"+plugin+"@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "安装 %s 失败: %v\n", plugin, err)
			os.Exit(1)
		}
		fmt.Printf("%s 已安装\n", plugin)
	} else {
		fmt.Printf("找到 %s: %s\n", plugin, path)
	}
}

func installPhantasmPlugins() {
	// 安装phantasm的protoc插件

	// protoc-gen-go-http
	fmt.Println("安装 protoc-gen-go-http...")
	cmd := exec.Command("go", "install", "github.com/dormoron/phantasm/cmd/protoc-gen-cosmos-http@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // 忽略错误，因为可能还没有这个包

	// protoc-gen-go-errors
	fmt.Println("安装 protoc-gen-go-errors...")
	cmd = exec.Command("go", "install", "github.com/dormoron/phantasm/cmd/protoc-gen-cosmos-errors@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // 忽略错误
}
