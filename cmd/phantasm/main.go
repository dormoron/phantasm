package main

import (
	"fmt"
	"log"

	"github.com/dormoron/phantasm"
	"github.com/dormoron/phantasm/cmd/phantasm/internal/project"
	"github.com/dormoron/phantasm/cmd/phantasm/internal/proto"
	"github.com/dormoron/phantasm/cmd/phantasm/internal/run"
	"github.com/dormoron/phantasm/cmd/phantasm/internal/upgrade"

	"github.com/spf13/cobra"
)

// rootCmd 是phantasm工具的根命令
var rootCmd = &cobra.Command{
	Use:     "phantasm",
	Short:   "Phantasm: 一个简洁、强大的Go微服务框架",
	Long:    `Phantasm是一个简洁、强大的Go微服务框架，集成了mist作为Web框架和eidola作为gRPC框架。`,
	Version: phantasm.VERSION,
}

// versionCmd 用于获取纯版本号
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本信息",
	Long:  `打印Phantasm工具的版本信息`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(phantasm.VERSION)
	},
}

func init() {
	rootCmd.AddCommand(project.CmdNew)
	rootCmd.AddCommand(proto.CmdProto)
	rootCmd.AddCommand(run.CmdRun)
	rootCmd.AddCommand(upgrade.CmdUpgrade)
	rootCmd.AddCommand(versionCmd)

	// 添加 -v 标志作为 --version 的别名
	rootCmd.Flags().BoolP("version", "v", false, "显示版本信息")
	// 覆盖默认的版本标志处理，使其只输出纯版本号
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
