package change

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// ReleaseInfo 保存发布信息结构
type ReleaseInfo struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
}

// CommitInfo 保存提交信息结构
type CommitInfo struct {
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

// ErrorInfo 保存错误信息结构
type ErrorInfo struct {
	Message string `json:"message"`
}

// GithubAPI 用于请求GitHub API的结构
type GithubAPI struct {
	Owner string
	Repo  string
	Token string
}

// GetReleaseInfo 获取phantasm发布信息
func (g *GithubAPI) GetReleaseInfo(version string) ReleaseInfo {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", g.Owner, g.Repo)
	if version != "latest" {
		api = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", g.Owner, g.Repo, version)
	}
	resp, code := requestGithubAPI(api, http.MethodGet, nil, g.Token)
	if code != http.StatusOK {
		printGithubErrorInfo(resp)
	}
	releaseInfo := ReleaseInfo{}
	err := json.Unmarshal(resp, &releaseInfo)
	if err != nil {
		fatal(err)
	}
	return releaseInfo
}

// GetCommitsInfo 获取phantasm提交信息
func (g *GithubAPI) GetCommitsInfo() []CommitInfo {
	info := g.GetReleaseInfo("latest")
	page := 1
	prePage := 100
	var list []CommitInfo
	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?pre_page=%d&page=%d&since=%s", g.Owner, g.Repo, prePage, page, info.PublishedAt)
		resp, code := requestGithubAPI(url, http.MethodGet, nil, g.Token)
		if code != http.StatusOK {
			printGithubErrorInfo(resp)
		}
		var res []CommitInfo
		err := json.Unmarshal(resp, &res)
		if err != nil {
			fatal(err)
		}
		list = append(list, res...)
		if len(res) < prePage {
			break
		}
		page++
	}
	return list
}

// 打印GitHub API错误信息
func printGithubErrorInfo(body []byte) {
	errorInfo := &ErrorInfo{}
	err := json.Unmarshal(body, errorInfo)
	if err != nil {
		fatal(err)
	}
	fatal(errors.New(errorInfo.Message))
}

// 请求GitHub API
func requestGithubAPI(url string, method string, body io.Reader, token string) ([]byte, int) {
	cli := &http.Client{Timeout: 60 * time.Second}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		fatal(err)
	}
	if token != "" {
		request.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	}
	resp, err := cli.Do(request)
	if err != nil {
		fatal(err)
	}
	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fatal(err)
	}
	return resBody, resp.StatusCode
}

// ParseCommitsInfo 解析提交信息生成Markdown
func ParseCommitsInfo(info []CommitInfo) string {
	group := map[string][]string{
		"fix":   {},
		"feat":  {},
		"deps":  {},
		"build": {},
		"break": {},
		"chore": {},
		"other": {},
	}

	for _, commitInfo := range info {
		msg := commitInfo.Commit.Message
		index := strings.Index(fmt.Sprintf("%q", msg), `\n`)
		if index != -1 {
			msg = msg[:index-1]
		}
		prefix := []string{"fix", "feat", "build", "deps", "break", "chore"}
		var matched bool
		for _, v := range prefix {
			msg = strings.TrimPrefix(msg, " ")
			if strings.HasPrefix(msg, v) {
				group[v] = append(group[v], msg)
				matched = true
				break
			}
		}
		if !matched {
			group["other"] = append(group["other"], msg)
		}
	}

	md := make(map[string]string)
	for key, value := range group {
		var text string
		switch key {
		case "break":
			text = "### 破坏性变更\n"
		case "deps":
			text = "### 依赖更新\n"
		case "feat":
			text = "### 新特性\n"
		case "fix":
			text = "### 缺陷修复\n"
		case "build":
			text = "### 构建系统\n"
		case "chore":
			text = "### 其他变更\n"
		case "other":
			text = "### 未分类变更\n"
		}
		if len(value) == 0 {
			continue
		}
		md[key] += text
		for _, value := range value {
			md[key] += fmt.Sprintf("- %s\n", value)
		}
	}
	return fmt.Sprint(md["break"], md["deps"], md["feat"], md["fix"], md["build"], md["chore"], md["other"])
}

// ParseReleaseInfo 解析发布信息生成文本
func ParseReleaseInfo(info ReleaseInfo) string {
	reg := regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z|<[\S\s]+?>`)
	body := reg.ReplaceAll([]byte(info.Body), []byte(""))
	if string(body) == "" {
		body = []byte("无发布信息")
	}
	splitters := "--------------------------------------------"
	return fmt.Sprintf(
		"作者: %s\n日期: %s\n链接: %s\n\n%s\n\n%s\n\n%s\n",
		info.Author.Login,
		info.PublishedAt,
		info.HTMLURL,
		splitters,
		body,
		splitters,
	)
}

// ParseGithubURL 解析GitHub URL获取所有者和仓库名
func ParseGithubURL(url string) (owner string, repo string) {
	var start int
	start = strings.Index(url, "//")
	if start == -1 {
		start = strings.Index(url, ":") + 1
	} else {
		start += 2
	}
	url = url[start:]
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		owner = parts[len(parts)-2]
		repo = parts[len(parts)-1]
	}
	return
}

// fatal 处理致命错误
func fatal(err error) {
	fmt.Fprintf(os.Stderr, "错误: %s\n", err.Error())
	os.Exit(1)
}
