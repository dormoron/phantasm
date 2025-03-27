package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// CmdUpgrade 表示升级命令
var CmdUpgrade = &cobra.Command{
	Use:   "upgrade",
	Short: "升级Phantasm工具",
	Long:  `升级Phantasm命令行工具到最新版本`,
	Run:   runUpgrade,
}

func runUpgrade(cmd *cobra.Command, args []string) {
	fmt.Println("正在检查当前版本...")
	currentVersion := getCurrentVersion()
	fmt.Printf("当前版本: %s\n", currentVersion)

	fmt.Println("正在检查最新版本...")
	latestVersion := getLatestVersion()
	fmt.Printf("最新版本: %s\n", latestVersion)

	if currentVersion == latestVersion {
		fmt.Println("您已经使用最新版本")
		return
	}

	fmt.Printf("正在从 %s 升级到 %s...\n", currentVersion, latestVersion)

	// 升级phantasm CLI工具
	goCmd := exec.Command("go", "install", "github.com/dormoron/phantasm/cmd/phantasm@latest")
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	if err := goCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "升级Phantasm工具失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Phantasm工具已成功升级到最新版本")
}

func getCurrentVersion() string {
	cmd := exec.Command("phantasm", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// 由于我们已经修改了版本模板，现在应该直接返回版本号
	return strings.TrimSpace(string(output))
}

func getLatestVersion() string {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", "github.com/dormoron/phantasm@latest")
	output, err := cmd.Output()
	if err != nil {
		// 尝试使用go mod查询
		altCmd := exec.Command("go", "mod", "download", "-json", "github.com/dormoron/phantasm@latest")
		if altOutput, altErr := altCmd.Output(); altErr == nil {
			// 从JSON输出中提取版本
			if strings.Contains(string(altOutput), "\"Version\":") {
				parts := strings.Split(string(altOutput), "\"Version\":")
				if len(parts) > 1 {
					versionPart := strings.Split(parts[1], ",")[0]
					return strings.Trim(strings.TrimSpace(versionPart), "\"")
				}
			}
		}
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
