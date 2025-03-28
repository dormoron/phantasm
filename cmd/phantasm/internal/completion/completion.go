package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CmdCompletion 表示自动补全命令
var CmdCompletion = &cobra.Command{
	Use:   "completion",
	Short: "生成命令行自动补全脚本",
	Long: `生成命令行自动补全脚本，支持 bash、zsh、fish 和 powershell。

使用方法：
  # 生成 bash 补全脚本
  phantasm completion bash > ~/.local/share/bash-completion/completions/phantasm

  # 生成 zsh 补全脚本
  phantasm completion zsh > ~/.zsh/completions/_phantasm

  # 生成 fish 补全脚本
  phantasm completion fish > ~/.config/fish/completions/phantasm.fish

  # 生成 powershell 补全脚本
  phantasm completion powershell > ~/.config/powershell/completions/phantasm.ps1`,
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "错误: 请指定 shell 类型 (bash|zsh|fish|powershell)\n")
		os.Exit(1)
	}

	shell := args[0]
	switch shell {
	case "bash":
		if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "生成 bash 补全脚本失败: %v\n", err)
			os.Exit(1)
		}
	case "zsh":
		if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "生成 zsh 补全脚本失败: %v\n", err)
			os.Exit(1)
		}
	case "fish":
		if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
			fmt.Fprintf(os.Stderr, "生成 fish 补全脚本失败: %v\n", err)
			os.Exit(1)
		}
	case "powershell":
		if err := cmd.Root().GenPowerShellCompletion(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "生成 powershell 补全脚本失败: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "错误: 不支持的 shell 类型 %s\n", shell)
		fmt.Fprintf(os.Stderr, "支持的 shell 类型: bash, zsh, fish, powershell\n")
		os.Exit(1)
	}
}
