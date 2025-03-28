package change

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CmdChange 是phantasm变更日志工具
var CmdChange = &cobra.Command{
	Use:   "changelog",
	Short: "获取phantasm变更日志",
	Long:  `获取phantasm发布版本或提交信息。例如：phantasm changelog dev 或 phantasm changelog {version}`,
	Run:   run,
}

var (
	token   string
	repoURL string
)

func init() {
	if repoURL = os.Getenv("PHANTASM_REPO"); repoURL == "" {
		repoURL = "https://github.com/dormoron/phantasm.git"
	}
	CmdChange.Flags().StringVarP(&repoURL, "repo-url", "r", repoURL, "github仓库")
	token = os.Getenv("GITHUB_TOKEN")
}

func run(_ *cobra.Command, args []string) {
	if token == "" {
		fmt.Fprintf(os.Stderr, "警告: 未设置 GITHUB_TOKEN 环境变量，可能会受到 GitHub API 速率限制\n")
		fmt.Fprintf(os.Stderr, "建议设置 GITHUB_TOKEN 环境变量以获得更好的体验\n")
		fmt.Fprintf(os.Stderr, "您可以在 https://github.com/settings/tokens 创建个人访问令牌\n\n")
	}

	owner, repo := ParseGithubURL(repoURL)
	api := GithubAPI{Owner: owner, Repo: repo, Token: token}
	version := "latest"
	if len(args) > 0 {
		version = args[0]
	}

	if version == "dev" {
		fmt.Println("正在获取开发版本提交信息...")
		info := api.GetCommitsInfo()
		if len(info) == 0 {
			fmt.Fprintf(os.Stderr, "未找到任何提交信息\n")
			os.Exit(1)
		}
		fmt.Print(ParseCommitsInfo(info))
		return
	}

	fmt.Printf("正在获取版本 %s 的发布信息...\n", version)
	info := api.GetReleaseInfo(version)
	if info.PublishedAt == "" {
		fmt.Fprintf(os.Stderr, "未找到指定版本的发布信息\n")
		os.Exit(1)
	}
	fmt.Print(ParseReleaseInfo(info))
}
