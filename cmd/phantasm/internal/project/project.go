package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CmdNew 表示创建新项目的命令
var CmdNew = &cobra.Command{
	Use:   "new [name]",
	Short: "创建一个新的项目",
	Long:  `创建一个包含必要文件和目录的新项目`,
	Run:   run,
}

var (
	repoURL     string
	moduleName  string
	withGrpc    bool
	withHttp    bool
	withDocker  bool
	withK8s     bool
	withGitHook bool
)

func init() {
	CmdNew.Flags().StringVarP(&repoURL, "repo-url", "r", "", "指定项目的版本控制系统URL")
	CmdNew.Flags().StringVarP(&moduleName, "module", "m", "", "指定Go模块名称")
	CmdNew.Flags().BoolVarP(&withGrpc, "grpc", "g", true, "是否包含gRPC服务器")
	CmdNew.Flags().BoolVarP(&withHttp, "http", "", true, "是否包含HTTP服务器")
	CmdNew.Flags().BoolVarP(&withDocker, "docker", "d", true, "是否包含Dockerfile")
	CmdNew.Flags().BoolVarP(&withK8s, "k8s", "k", false, "是否包含Kubernetes配置")
	CmdNew.Flags().BoolVarP(&withGitHook, "git-hook", "", false, "是否安装Git钩子")
}

func run(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "必须指定项目名称")
		os.Exit(1)
	}
	name := args[0]
	if moduleName == "" {
		moduleName = name
		if repoURL != "" {
			moduleName = repoURL
			if !strings.HasPrefix(moduleName, "github.com/") && !strings.HasPrefix(moduleName, "gitlab.com/") {
				if strings.HasPrefix(moduleName, "http://") || strings.HasPrefix(moduleName, "https://") {
					fmt.Fprintln(os.Stderr, "模块名称不应以http://或https://开头")
					os.Exit(1)
				}
				moduleName = strings.TrimSuffix(repoURL, ".git")
				if strings.HasPrefix(moduleName, "git@") {
					fields := strings.Split(moduleName, ":")
					if len(fields) == 2 {
						moduleName = strings.Replace(fields[0], "git@", "", 1) + "/" + fields[1]
					}
				}
			}
		}
	}

	fmt.Printf("创建项目: %s\n", name)
	fmt.Printf("模块名: %s\n", moduleName)

	if err := createProject(name, moduleName); err != nil {
		fmt.Fprintf(os.Stderr, "创建项目失败: %v\n", err)
		os.Exit(1)
	}
}

func createProject(name, module string) error {
	// 创建项目根目录
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}

	// 创建目录结构
	dirs := []string{
		"api",
		"cmd/" + name,
		"configs",
		"internal/server",
		"internal/service",
		"internal/data",
		"internal/biz",
		"internal/conf",
		"third_party",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(name, dir), 0755); err != nil {
			return err
		}
	}

	// 创建go.mod文件
	gomod := fmt.Sprintf("module %s\n\ngo 1.22\n", module)
	if err := os.WriteFile(filepath.Join(name, "go.mod"), []byte(gomod), 0644); err != nil {
		return err
	}

	// 创建主程序文件
	mainContent := fmt.Sprintf(`package main

import (
	"flag"
	"os"

	"%s"
	"%s/config"
	"%s/log"
	%s
	%s
	
	"go.uber.org/zap"
)

var (
	// Name 是应用程序名称
	Name = "%s"
	// Version 是应用程序版本
	Version = "v1.0.0"
	// flagconf 是配置路径
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()

	// 初始化logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zlog := log.NewZapLogger(logger)
	
	// 创建应用程序
	app := %s.New(
		%s.Name(Name),
		%s.Version(Version),
		%s.Logger(zlog),
		%s,
		%s
	)
	
	// 启动应用程序
	if err := app.Run(); err != nil {
		zlog.Fatal(err.Error())
		os.Exit(1)
	}
}
`,
		module,
		module,
		module,
		getHttpImport(withHttp, module),
		getGrpcImport(withGrpc, module),
		name,
		module,
		module,
		module,
		module,
		getHttpServer(withHttp, module),
		getGrpcServer(withGrpc, module),
	)

	if err := os.WriteFile(filepath.Join(name, "cmd", name, "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// 创建服务器初始化文件
	if withHttp {
		httpContent := fmt.Sprintf(`package main

import (
	"%s/log"
	"%s/transport/http"
	
	"github.com/dormoron/mist"
)

func newHTTPServer(logger log.Logger) *http.Server {
	engine := mist.Default()
	
	// 注册路由
	engine.GET("/", func(c *mist.Context) {
		c.String(200, "Hello World!")
	})
	
	srv := http.NewServer(
		http.Address(":8000"),
		http.Logger(logger),
		http.Engine(engine),
	)
	
	return srv
}
`,
			module,
			module,
		)
		if err := os.WriteFile(filepath.Join(name, "cmd", name, "http.go"), []byte(httpContent), 0644); err != nil {
			return err
		}
	}

	if withGrpc {
		grpcContent := fmt.Sprintf(`package main

import (
	"%s/log"
	"%s/transport/grpc"
	
	"github.com/dormoron/eidola"
)

func newGRPCServer(logger log.Logger) *grpc.Server {
	srv := grpc.NewServer(
		grpc.Address(":9000"),
		grpc.Logger(logger),
	)
	
	return srv
}
`,
			module,
			module,
		)
		if err := os.WriteFile(filepath.Join(name, "cmd", name, "grpc.go"), []byte(grpcContent), 0644); err != nil {
			return err
		}
	}

	// 创建配置文件
	configContent := `server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
data:
  database:
    driver: mysql
    source: root:password@tcp(127.0.0.1:3306)/test
  redis:
    addr: 127.0.0.1:6379
    read_timeout: 0.2s
    write_timeout: 0.2s
`
	if err := os.WriteFile(filepath.Join(name, "configs", "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	// 创建README.md
	readmeContent := fmt.Sprintf(`# %s

基于Phantasm框架构建的微服务项目

## 目录结构

- api: API定义
- cmd: 应用程序入口
- configs: 配置文件
- internal: 内部代码
  - biz: 业务逻辑
  - data: 数据访问
  - server: 服务器初始化
  - service: 服务实现
- third_party: 第三方代码

## 快速开始

### 运行

%s

### 构建

%s

## 配置

配置文件位于 configs/config.yaml
`,
		name,
		fmt.Sprintf("```bash\ngo run ./cmd/%s\n```", name),
		fmt.Sprintf("```bash\ngo build -o bin/%s ./cmd/%s\n```", name, name),
	)

	if err := os.WriteFile(filepath.Join(name, "README.md"), []byte(readmeContent), 0644); err != nil {
		return err
	}

	if withDocker {
		dockerContent := `FROM golang:1.22 AS builder

WORKDIR /src
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /app ./cmd/` + name + `

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app /app/
COPY --from=builder /src/configs /app/configs

EXPOSE 8000 9000
ENTRYPOINT ["/app/` + name + `"]
`
		if err := os.WriteFile(filepath.Join(name, "Dockerfile"), []byte(dockerContent), 0644); err != nil {
			return err
		}
	}

	if withK8s {
		k8sContent := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        ports:
        - containerPort: 8000
        - containerPort: 9000
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: %s
  ports:
  - name: http
    port: 8000
    targetPort: 8000
  - name: grpc
    port: 9000
    targetPort: 9000
`, name, name, name, name, name, name, name)
		if err := os.WriteFile(filepath.Join(name, "deploy", "kubernetes", name+".yaml"), []byte(k8sContent), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("项目 %s 创建成功!\n", name)
	return nil
}

// 添加辅助函数
func getHttpImport(withHttp bool, module string) string {
	if withHttp {
		return fmt.Sprintf(`"%s/transport/http"`, module)
	}
	return ""
}

func getGrpcImport(withGrpc bool, module string) string {
	if withGrpc {
		return fmt.Sprintf(`"%s/transport/grpc"`, module)
	}
	return ""
}

func getHttpServer(withHttp bool, module string) string {
	if withHttp {
		return fmt.Sprintf(`%s.Server(newHTTPServer(zlog))`, module)
	}
	return ""
}

func getGrpcServer(withGrpc bool, module string) string {
	if withGrpc {
		return fmt.Sprintf(`%s.Server(newGRPCServer(zlog))`, module)
	}
	return ""
}
