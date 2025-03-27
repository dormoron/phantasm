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
	Long:  "获取phantasm发布版本或提交信息。例如：phantasm changelog dev 或 phantasm changelog {version}",
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
	owner, repo := ParseGithubURL(repoURL)
	api := GithubAPI{Owner: owner, Repo: repo, Token: token}
	version := "latest"
	if len(args) > 0 {
		version = args[0]
	}
	if version == "dev" {
		info := api.GetCommitsInfo()
		fmt.Print(ParseCommitsInfo(info))
		return
	}
	info := api.GetReleaseInfo(version)
	fmt.Print(ParseReleaseInfo(info))
}
