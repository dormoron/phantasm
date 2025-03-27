package run

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CmdRun 表示运行项目的命令
var CmdRun = &cobra.Command{
	Use:   "run",
	Short: "运行项目",
	Long:  `编译并运行项目`,
	Run:   runProject,
}

var (
	buildFlags string
	runArgs    string
)

func init() {
	CmdRun.Flags().StringVarP(&buildFlags, "build-flags", "b", "", "传递给go build的标志")
	CmdRun.Flags().StringVarP(&runArgs, "args", "a", "", "传递给应用程序的命令行参数")
}

func runProject(cmd *cobra.Command, args []string) {
	var mainFile string
	var cmdPath string

	// 查找入口文件
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && strings.Contains(path, "cmd") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if strings.Contains(string(content), "func main()") {
				mainFile = path
				cmdPath = filepath.Dir(path)
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "查找入口文件时出错: %v\n", err)
		os.Exit(1)
	}

	if mainFile == "" {
		fmt.Fprintln(os.Stderr, "未找到主入口文件")
		os.Exit(1)
	}

	fmt.Printf("找到入口文件: %s\n", mainFile)

	// 构建运行命令
	goRun := exec.Command("go", "run")

	// 添加构建标志
	if buildFlags != "" {
		flags := strings.Split(buildFlags, " ")
		for _, flag := range flags {
			if flag != "" {
				goRun.Args = append(goRun.Args, flag)
			}
		}
	}

	// 添加主文件路径
	goRun.Args = append(goRun.Args, cmdPath)

	// 添加应用程序参数
	if runArgs != "" {
		args := strings.Split(runArgs, " ")
		for _, arg := range args {
			if arg != "" {
				goRun.Args = append(goRun.Args, arg)
			}
		}
	}

	// 设置命令输出到标准输出和标准错误
	goRun.Stdout = os.Stdout
	goRun.Stderr = os.Stderr

	fmt.Printf("运行命令: %s\n", strings.Join(goRun.Args, " "))

	// 执行命令
	if err := goRun.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "运行项目失败: %v\n", err)
		os.Exit(1)
	}
}
