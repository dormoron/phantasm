package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CmdNew 表示创建新项目的命令
var CmdNew = &cobra.Command{
	Use:   "new [name]",
	Short: "创建一个新的项目",
	Long:  `创建一个包含必要文件和目录的新项目，集成mist作为Web框架和eidola作为gRPC框架`,
	Run:   run,
}

var (
	repoURL     string
	branch      string
	timeout     string
	moduleName  string
	withGrpc    bool
	withHttp    bool
	withDocker  bool
	withK8s     bool
	withGitHook bool
)

func init() {
	if repoURL = os.Getenv("PHANTASM_LAYOUT_REPO"); repoURL == "" {
		repoURL = "https://github.com/dormoron/phantasm-layout.git"
	}
	timeout = "60s"
	CmdNew.Flags().StringVarP(&repoURL, "repo-url", "r", repoURL, "项目模板仓库")
	CmdNew.Flags().StringVarP(&branch, "branch", "b", branch, "仓库分支")
	CmdNew.Flags().StringVarP(&timeout, "timeout", "t", timeout, "超时时间")
	CmdNew.Flags().StringVarP(&moduleName, "module", "m", "", "指定Go模块名称")
	CmdNew.Flags().BoolVarP(&withGrpc, "grpc", "g", true, "是否包含gRPC服务器")
	CmdNew.Flags().BoolVarP(&withHttp, "http", "", true, "是否包含HTTP服务器")
	CmdNew.Flags().BoolVarP(&withDocker, "docker", "d", true, "是否包含Dockerfile")
	CmdNew.Flags().BoolVarP(&withK8s, "k8s", "k", false, "是否包含Kubernetes配置")
	CmdNew.Flags().BoolVarP(&withGitHook, "git-hook", "", false, "是否安装Git钩子")
}

func run(cmd *cobra.Command, args []string) {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前工作目录失败: %v\n", err)
		os.Exit(1)
	}

	// 解析超时时间
	t, err := time.ParseDuration(timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析超时时间失败: %v\n", err)
		os.Exit(1)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	// 获取项目名称
	name := ""
	if len(args) == 0 {
		prompt := &survey.Input{
			Message: "请输入项目名称:",
			Help:    "将创建的项目名称。",
		}
		err = survey.AskOne(prompt, &name)
		if err != nil || name == "" {
			fmt.Println("项目名称不能为空")
			return
		}
	} else {
		name = args[0]
	}

	// 处理项目路径
	projectName, workingDir := processProjectParams(name, wd)

	// 如果没有指定模块名称，使用项目名称
	if moduleName == "" {
		// 提示用户输入模块名称
		modulePrompt := &survey.Input{
			Message: "请输入Go模块名称:",
			Default: projectName,
			Help:    "Go模块名称，通常是项目的仓库路径，如github.com/username/project",
		}
		err = survey.AskOne(modulePrompt, &moduleName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取模块名称失败: %v\n", err)
			return
		}
	}

	// 确认是否包含gRPC和HTTP服务
	if !cmd.Flags().Changed("grpc") {
		grpcPrompt := &survey.Confirm{
			Message: "是否包含gRPC服务?",
			Default: true,
			Help:    "如果选择是，将使用eidola框架创建gRPC服务",
		}
		survey.AskOne(grpcPrompt, &withGrpc)
	}

	if !cmd.Flags().Changed("http") {
		httpPrompt := &survey.Confirm{
			Message: "是否包含HTTP服务?",
			Default: true,
			Help:    "如果选择是，将使用mist框架创建HTTP服务",
		}
		survey.AskOne(httpPrompt, &withHttp)
	}

	// 创建项目
	fmt.Printf("创建项目: %s\n", projectName)
	fmt.Printf("模块名: %s\n", moduleName)
	fmt.Printf("包含gRPC服务: %v\n", withGrpc)
	fmt.Printf("包含HTTP服务: %v\n", withHttp)

	// 如果模板仓库可用，尝试使用模板创建项目
	if useTemplateRepo() {
		p := &Project{Name: projectName}
		done := make(chan error, 1)
		go func() {
			done <- p.New(ctx, workingDir, repoURL, branch, moduleName, withGrpc, withHttp, withDocker, withK8s)
		}()

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				fmt.Fprint(os.Stderr, "\033[31m错误: 项目创建超时\033[m\n")
				return
			}
			fmt.Fprintf(os.Stderr, "\033[31m错误: 创建项目失败(%s)\033[m\n", ctx.Err().Error())
		case err = <-done:
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m错误: 创建项目失败(%s)\033[m\n", err.Error())
			}
		}
		return
	}

	// 如果模板仓库不可用，使用本地生成项目
	if err := createProject(projectName, moduleName, workingDir); err != nil {
		fmt.Fprintf(os.Stderr, "创建项目失败: %v\n", err)
		os.Exit(1)
	}
}

// 处理项目参数
func processProjectParams(projectName string, workingDir string) (projectNameResult, workingDirResult string) {
	_projectDir := projectName
	_workingDir := workingDir

	// 处理带有系统变量的项目名称
	if strings.HasPrefix(projectName, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return _projectDir, _workingDir
		}
		_projectDir = filepath.Join(homeDir, projectName[2:])
	}

	// 检查路径是否为相对路径
	if !filepath.IsAbs(projectName) {
		absPath, err := filepath.Abs(projectName)
		if err != nil {
			return _projectDir, _workingDir
		}
		_projectDir = absPath
	}

	return filepath.Base(_projectDir), filepath.Dir(_projectDir)
}

// 检查是否使用模板仓库
func useTemplateRepo() bool {
	// 这里可以添加检查模板仓库是否可用的逻辑
	// 简单起见，先返回false，使用本地生成项目
	return false
}

// Project 是项目模板
type Project struct {
	Name string
	Path string
}

// New 从远程仓库创建一个新项目
func (p *Project) New(ctx context.Context, dir string, layout string, branch string, moduleName string, withGrpc bool, withHttp bool, withDocker bool, withK8s bool) error {
	// 这里实现从远程仓库创建项目的逻辑
	// 由于这部分需要实现克隆仓库等复杂逻辑，先留空
	return nil
}

// 创建项目的本地实现
func createProject(name, module, workingDir string) error {
	// 创建项目根目录
	projectPath := filepath.Join(workingDir, name)
	fmt.Printf("创建项目目录: %s\n", projectPath)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		fmt.Printf("创建项目目录失败: %v\n", err)
		return err
	}

	// 创建目录结构
	dirs := []string{
		"api",
		"cmd/" + name,
		"configs",
		"internal/biz",
		"internal/conf",
		"internal/data",
		"internal/server",
		"internal/service",
		"internal/pkg/middleware",
		"internal/pkg/errorx",
		"third_party/google/api",
		"third_party/validate",
		"third_party/openapi",
		"third_party/errors",
		"scripts",
	}

	if withK8s {
		dirs = append(dirs, "deploy/kubernetes")
	}

	fmt.Println("创建项目目录结构...")
	for _, dir := range dirs {
		fullPath := filepath.Join(projectPath, dir)
		fmt.Printf("  创建目录: %s\n", dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			fmt.Printf("创建目录 %s 失败: %v\n", fullPath, err)
			return err
		}
	}

	// 创建go.mod文件
	gomod := fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire (\n\tgithub.com/dormoron/eidola v0.1.0\n\tgithub.com/dormoron/mist v0.1.0\n\tgithub.com/dormoron/phantasm v0.1.0\n\tgo.uber.org/zap v1.26.0\n)\n", module)
	if err := writeUTF8File(filepath.Join(projectPath, "go.mod"), []byte(gomod), 0644); err != nil {
		return err
	}

	// 创建main.go文件
	mainContent := fmt.Sprintf(`package main

import (
	"flag"
	"os"

	"github.com/dormoron/phantasm"
	"github.com/dormoron/phantasm/config"
	"github.com/dormoron/phantasm/log"
	%s
	%s
	
	"%s/internal/conf"
	"%s/internal/server"
	"%s/internal/service"
	
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
	
	// 加载配置
	c := config.New(
		config.WithSource(
			config.NewFileSource(flagconf),
		),
	)
	if err := c.Load(); err != nil {
		zlog.Fatal(err.Error())
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		zlog.Fatal(err.Error())
	}
	
	// 创建应用程序
	app := phantasm.New(
		phantasm.Name(Name),
		phantasm.Version(Version),
		phantasm.Logger(zlog),
		phantasm.Server(
			%s
			%s
		),
	)
	
	// 启动应用程序
	if err := app.Run(); err != nil {
		zlog.Fatal(err.Error())
		os.Exit(1)
	}
}
`,
		getHttpImport(withHttp),
		getGrpcImport(withGrpc),
		module,
		module,
		module,
		name,
		getServerInit(withHttp, "server.NewHTTPServer(&bc.Server, zlog, service.New(zlog))"),
		getServerInit(withGrpc, "server.NewGRPCServer(&bc.Server, zlog, service.New(zlog))"),
	)

	if err := writeUTF8File(filepath.Join(projectPath, "cmd", name, "main.go"), []byte(mainContent), 0644); err != nil {
		return err
	}

	// 创建internal/server下的文件
	// HTTP服务器
	if withHttp {
		httpServerContent := fmt.Sprintf(`package server

import (
    "github.com/dormoron/mist"
    "github.com/dormoron/phantasm/log"
    "github.com/dormoron/phantasm/middleware/logging"
    "github.com/dormoron/phantasm/middleware/recovery"
    "github.com/dormoron/phantasm/transport/http"
    
    "%s/internal/conf"
    "%s/internal/service"
)

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(c *conf.Server, logger log.Logger, svc *service.Service) *http.Server {
    var opts = []http.ServerOption{
        http.Address(c.Http.Addr),
        http.Timeout(c.Http.Timeout.AsDuration()),
        http.Logger(logger),
    }
    
    srv := http.NewServer(opts...)
    
    // 创建Mist引擎并设置中间件
    mServer, err := http.NewHTTPServer(
        http.WithAddress(c.Http.Addr),
        http.WithTimeout(c.Http.Timeout.AsDuration()),
    )
    if err != nil {
        panic(err)
    }
    
    // 使用中间件
    mServer.UseMiddleware(
        recovery.Recovery(),
        logging.Logging(
            logging.WithLogger(logger),
            logging.WithLogRequestBody(true),
            logging.WithLogResponseBody(true),
        ),
    )
    
    // 注册API路由组
    api := mServer.Group("/api")
    {
        v1 := api.Group("/v1")
        {
            v1.GET("/hello/:name", func(c *mist.Context) {
                nameVal, err := c.PathValue("name").String()
                if err != nil {
                    c.RespondWithJSON(400, map[string]string{"error": "无效的名称参数"})
                    return
                }
                // 调用服务实现
                message := "Hello " + nameVal
                c.RespondWithJSON(200, map[string]interface{}{
                    "message": message,
                })
            })
        }
    }
    
    // 健康检查
    mServer.GET("/health", func(c *mist.Context) {
        c.RespondWithJSON(200, map[string]string{"status": "ok"})
    })
    
    return srv
}`, module, module)

		if err := writeUTF8File(filepath.Join(projectPath, "internal", "server", "http.go"), []byte(httpServerContent), 0644); err != nil {
			return err
		}
	}

	// gRPC服务器
	if withGrpc {
		grpcServerContent := fmt.Sprintf(`package server

import (
    "github.com/dormoron/phantasm/log"
    "github.com/dormoron/phantasm/middleware/logging"
    "github.com/dormoron/phantasm/middleware/recovery"
    "github.com/dormoron/phantasm/transport/grpc"
    
    "%s/internal/conf"
    "%s/internal/service"
    
    v1 "%s/api/%s/v1"
)

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(c *conf.Server, logger log.Logger, svc *service.Service) *grpc.Server {
    // 创建gRPC服务器
    server := grpc.NewServer(
        grpc.Address(c.Grpc.Addr),
        grpc.Timeout(c.Grpc.Timeout.AsDuration()),
        grpc.Logger(logger),
        grpc.Name("%s-service"),
    )
    
    // 使用中间件
    server.UseMiddleware(
        recovery.Recovery(),
        logging.Logging(
            logging.WithLogger(logger),
            logging.WithLogRequestBody(true),
            logging.WithLogResponseBody(true),
        ),
    )
    
    // 注册服务
    v1.Register%sServer(server, svc)
    
    return server
}`, module, module, module, name, name, strings.Title(name))

		if err := writeUTF8File(filepath.Join(projectPath, "internal", "server", "server.go"), []byte(grpcServerContent), 0644); err != nil {
			return err
		}
	}

	// 创建internal/service下的服务实现
	serviceContent := fmt.Sprintf(`package service

import (
	"context"

	"github.com/dormoron/phantasm/log"
	
	v1 "%s/api/%s/v1"
	"%s/internal/biz"
)

// Service 是实现所有服务端点的服务对象
type Service struct {
	v1.Unimplemented%sServer

	log  log.Logger
	greeter *biz.GreeterUsecase
}

// New 创建Service实例
func New(logger log.Logger) *Service {
	return &Service{
		log:  logger,
		greeter: biz.NewGreeterUsecase(logger),
	}
}

// SayHello 实现了v1.GreeterServer接口
func (s *Service) SayHello(ctx context.Context, req *v1.HelloRequest) (*v1.HelloReply, error) {
	s.log.WithContext(ctx).Infof("SayHello Received: %%s", req.GetName())
	return &v1.HelloReply{Message: "Hello " + req.GetName()}, nil
}
`, module, name, module, strings.Title(name))

	if err := writeUTF8File(filepath.Join(projectPath, "internal", "service", "service.go"), []byte(serviceContent), 0644); err != nil {
		return err
	}

	// 创建internal/biz业务逻辑
	bizContent := fmt.Sprintf(`package biz

import (
	"github.com/dormoron/phantasm/log"
)

// GreeterUsecase 是问候语的业务逻辑用例
type GreeterUsecase struct {
	log log.Logger
}

// NewGreeterUsecase 创建GreeterUsecase实例
func NewGreeterUsecase(logger log.Logger) *GreeterUsecase {
	return &GreeterUsecase{log: logger}
}

// 为GreeterUsecase目录创建README
func init() {
	// 此层为业务逻辑层，类似领域层
	// 定义领域对象及其业务行为
}
`)

	if err := writeUTF8File(filepath.Join(projectPath, "internal", "biz", "greeter.go"), []byte(bizContent), 0644); err != nil {
		return err
	}

	// 创建internal/biz/README.md
	bizReadme := `# Biz

业务逻辑层，类似DDD中的领域层（Domain Layer）

此层主要包含：
1. 领域对象（Domain Object）及其业务行为
2. 领域服务（Domain Service）
3. 仓储接口（Repository Interface）
`
	if err := writeUTF8File(filepath.Join(projectPath, "internal", "biz", "README.md"), []byte(bizReadme), 0644); err != nil {
		return err
	}

	// 创建internal/data数据层
	dataContent := fmt.Sprintf(`package data

import (
	"github.com/dormoron/phantasm/log"

	"%s/internal/biz"
)

// 数据访问层
// 实现biz层定义的仓储接口

// Data 包含所有数据源的客户端实例
type Data struct {
	log log.Logger
}

// NewData 创建Data实例
func NewData(logger log.Logger) (*Data, error) {
	return &Data{
		log: logger,
	}, nil
}
`, module)

	if err := writeUTF8File(filepath.Join(projectPath, "internal", "data", "data.go"), []byte(dataContent), 0644); err != nil {
		return err
	}

	// 创建internal/data/README.md
	dataReadme := `# Data

数据访问层，主要负责与各种外部数据源交互

此层主要包含：
1. 数据库访问逻辑
2. 缓存访问逻辑
3. 外部服务调用
4. 实现biz层定义的仓储接口
`
	if err := writeUTF8File(filepath.Join(projectPath, "internal", "data", "README.md"), []byte(dataReadme), 0644); err != nil {
		return err
	}

	// 创建internal/conf配置
	confProtoContent := `syntax = "proto3";

package conf;

option go_package = "internal/conf;conf";

import "google/protobuf/duration.proto";

message Bootstrap {
  Server server = 1;
  Data data = 2;
}

message Server {
  message HTTP {
    string addr = 1;
    google.protobuf.Duration timeout = 2;
  }
  message GRPC {
    string addr = 1;
    google.protobuf.Duration timeout = 2;
  }
  HTTP http = 1;
  GRPC grpc = 2;
}

message Data {
  message Database {
    string driver = 1;
    string source = 2;
  }
  message Redis {
    string addr = 1;
    google.protobuf.Duration read_timeout = 2;
    google.protobuf.Duration write_timeout = 3;
  }
  Database database = 1;
  Redis redis = 2;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "internal", "conf", "conf.proto"), []byte(confProtoContent), 0644); err != nil {
		return err
	}

	// 创建conf.pb.go（实际项目中需要通过protoc生成）
	confPbContent := fmt.Sprintf(`package conf

// 实际项目中应该通过protoc命令生成此文件
// 为了简化示例，这里手动创建一个基本实现

import (
	"time"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Bootstrap 是应用程序的主要配置
type Bootstrap struct {
	Server *Server
	Data   *Data
}

// Server 包含服务器配置
type Server struct {
	Http *Server_HTTP
	Grpc *Server_GRPC
}

// Server_HTTP 包含HTTP服务器配置
type Server_HTTP struct {
	Addr    string
	Timeout *durationpb.Duration
}

// Server_GRPC 包含gRPC服务器配置
type Server_GRPC struct {
	Addr    string
	Timeout *durationpb.Duration
}

// Data 包含数据源配置
type Data struct {
	Database *Data_Database
	Redis    *Data_Redis
}

// Data_Database 包含数据库配置
type Data_Database struct {
	Driver string
	Source string
}

// Data_Redis 包含Redis配置
type Data_Redis struct {
	Addr         string
	ReadTimeout  *durationpb.Duration
	WriteTimeout *durationpb.Duration
}

// AsDuration 将Duration转换为time.Duration
func (d *durationpb.Duration) AsDuration() time.Duration {
	if d == nil {
		return 0
	}
	return time.Duration(d.Seconds) * time.Second + time.Duration(d.Nanos) * time.Nanosecond
}
`)

	if err := writeUTF8File(filepath.Join(projectPath, "internal", "conf", "conf.pb.go"), []byte(confPbContent), 0644); err != nil {
		return err
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
	if err := writeUTF8File(filepath.Join(projectPath, "configs", "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	// 创建third_party下的文件
	// Google API proto
	googleAPIHttpProto := `syntax = "proto3";

package google.api;

option go_package = "google.golang.org/genproto/googleapis/api/annotations;annotations";

import "google/protobuf/descriptor.proto";

extend google.protobuf.MethodOptions {
  // 与HTTP绑定的选项
  HttpRule http = 72295728;
}

// HttpRule 定义了API方法如何映射到HTTP REST API
message HttpRule {
  // 必须指定选择器之一
  oneof pattern {
    // 用于GET方法
    string get = 2;
    // 用于PUT方法
    string put = 3;
    // 用于POST方法
    string post = 4;
    // 用于DELETE方法
    string delete = 5;
    // 用于PATCH方法
    string patch = 6;
    // 自定义模式
    CustomPattern custom = 8;
  }

  // HTTP请求体
  string body = 7;

  // 附加绑定
  repeated HttpRule additional_bindings = 11;
}

// 自定义HTTP方法模式
message CustomPattern {
  // HTTP方法名称
  string kind = 1;
  // HTTP URL模板
  string path = 2;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "google", "api", "http.proto"), []byte(googleAPIHttpProto), 0644); err != nil {
		return err
	}

	googleAPIAnnotationsProto := `syntax = "proto3";

package google.api;

option go_package = "google.golang.org/genproto/googleapis/api/annotations;annotations";

import "google/api/http.proto";
import "google/protobuf/descriptor.proto";

extend google.protobuf.MethodOptions {
  // 查看 HttpRule 定义了解详情
  HttpRule http = 72295728;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "google", "api", "annotations.proto"), []byte(googleAPIAnnotationsProto), 0644); err != nil {
		return err
	}

	// 添加httpbody.proto
	httpbodyProto := `syntax = "proto3";

package google.api;

option go_package = "google.golang.org/genproto/googleapis/api/httpbody;httpbody";

import "google/protobuf/any.proto";

// HttpBody 消息型用于HTTP API请求和响应中包含任意媒体类型数据
message HttpBody {
  // HTTP Content-Type 头信息的媒体类型，表示内容的MIME类型
  string content_type = 1;

  // HTTP body的二进制数据
  bytes data = 2;

  // 与此消息关联的应用特定的附加信息
  repeated google.protobuf.Any extensions = 3;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "google", "api", "httpbody.proto"), []byte(httpbodyProto), 0644); err != nil {
		return err
	}

	// 添加validate.proto和相关README
	validateProto := `syntax = "proto3";

package validate;

option go_package = "github.com/envoyproxy/protoc-gen-validate/validate;validate";

import "google/protobuf/descriptor.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/timestamp.proto";

// 字段校验规则
extend google.protobuf.FieldOptions {
  FieldRules rules = 1071;
}

// 验证规则容器
message FieldRules {
  // 标量类型的验证规则
  oneof type {
    // 字符串类型的规则
    StringRules string = 1;
    // 数值类型的规则
    UInt32Rules uint32 = 2;
    UInt64Rules uint64 = 3;
    Int32Rules int32 = 4;
    Int64Rules int64 = 5;
    DoubleRules double = 6;
    FloatRules float = 7;
    BoolRules bool = 8;
    // 时间戳类型的规则
    TimestampRules timestamp = 9;
    // 时长类型的规则
    DurationRules duration = 10;
  }

  // 重复字段的规则
  RepeatedRules repeated = 11;
  // Map字段的规则
  MapRules map = 12;
  // 任意字段的规则
  AnyRules any = 13;
}

// 字符串规则
message StringRules {
  // 最小长度
  uint64 min_len = 1;
  // 最大长度
  uint64 max_len = 2;
  // 固定长度
  uint64 len = 3;
  // 匹配正则表达式
  string pattern = 4;
  // 前缀要求
  string prefix = 5;
  // 后缀要求
  string suffix = 6;
  // 是否包含
  string contains = 7;
  // 必须是有效的Email格式
  bool email = 8;
  // 必须是有效的主机名格式
  bool hostname = 9;
  // 必须是有效的IP地址
  bool ip = 10;
  // 必须是有效的IPv4地址
  bool ipv4 = 11;
  // 必须是有效的IPv6地址
  bool ipv6 = 12;
  // 必须是有效的URI
  bool uri = 13;
  // 必须是有效的URI引用
  bool uri_ref = 14;
}

// 整数规则
message UInt32Rules {
  uint32 const = 1;
  uint32 lt = 2;
  uint32 lte = 3;
  uint32 gt = 4;
  uint32 gte = 5;
}

message UInt64Rules {
  uint64 const = 1;
  uint64 lt = 2;
  uint64 lte = 3;
  uint64 gt = 4;
  uint64 gte = 5;
}

message Int32Rules {
  int32 const = 1;
  int32 lt = 2;
  int32 lte = 3;
  int32 gt = 4;
  int32 gte = 5;
}

message Int64Rules {
  int64 const = 1;
  int64 lt = 2;
  int64 lte = 3;
  int64 gt = 4;
  int64 gte = 5;
}

// 浮点数规则
message DoubleRules {
  double const = 1;
  double lt = 2;
  double lte = 3;
  double gt = 4;
  double gte = 5;
}

message FloatRules {
  float const = 1;
  float lt = 2;
  float lte = 3;
  float gt = 4;
  float gte = 5;
}

// 布尔规则
message BoolRules {
  bool const = 1;
}

// 时间戳规则
message TimestampRules {
  google.protobuf.Timestamp const = 1;
  google.protobuf.Timestamp lt = 2;
  google.protobuf.Timestamp lte = 3;
  google.protobuf.Timestamp gt = 4;
  google.protobuf.Timestamp gte = 5;
}

// 时长规则
message DurationRules {
  google.protobuf.Duration const = 1;
  google.protobuf.Duration lt = 2;
  google.protobuf.Duration lte = 3;
  google.protobuf.Duration gt = 4;
  google.protobuf.Duration gte = 5;
}

// 重复字段规则
message RepeatedRules {
  uint64 min_items = 1;
  uint64 max_items = 2;
  uint64 items = 3;
  bool unique = 4;
}

// Map字段规则
message MapRules {
  uint64 min_pairs = 1;
  uint64 max_pairs = 2;
}

// 任意消息规则
message AnyRules {
  repeated string in = 1;
  repeated string not_in = 2;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "validate", "validate.proto"), []byte(validateProto), 0644); err != nil {
		return err
	}

	validateReadme := `# Protocol Buffers Validation

这个目录包含用于协议缓冲区字段验证的proto文件。

## 验证规则

您可以使用这些验证规则来为您的proto消息字段添加约束：

- 字符串：长度、正则表达式匹配、邮件格式等
- 数字：范围约束、常量值等
- 重复字段：项目数量、唯一性等
- 其他规则：时间戳、持续时间等

## 如何使用

在您的proto文件中导入并使用validate包：

` + "```proto" + `
import "validate/validate.proto";

message YourMessage {
  string email = 1 [(validate.rules).string.email = true];
  int32 age = 2 [(validate.rules).int32 = {gt: 0, lt: 150}];
}
` + "```" + `

这将帮助您确保传入的数据符合您的业务规则。
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "validate", "README.md"), []byte(validateReadme), 0644); err != nil {
		return err
	}

	// 改进third_party的README文件
	thirdPartyReadme := `# Third Party Proto Files

这个目录包含第三方proto文件，用于生成API定义。这些proto文件用于支持REST API、参数验证等功能。

## 目录结构

- **google/api/**：包含Google API proto文件，用于HTTP REST API定义
  - annotations.proto：定义HTTP路由注解
  - http.proto：定义HTTP路由规则
  - httpbody.proto：支持HTTP body的自定义内容类型
  
- **validate/**：包含用于字段验证的proto文件
  - validate.proto：定义字段验证规则，如字符串长度、数字范围、格式等

- **openapi/**：包含OpenAPI生成相关文件
  - annotations.proto：定义OpenAPI元数据注解
  - openapi.proto：OpenAPI文档生成配置

- **errors/**：标准错误定义
  - errors.proto：定义统一的错误响应格式

## 使用方法

在您的proto文件中导入并使用这些定义：

` + "```protobuf" + `
syntax = "proto3";

package api.example.v1;

import "google/api/annotations.proto";
import "validate/validate.proto";
import "openapi/annotations.proto";
import "errors/errors.proto";

service ExampleService {
  rpc GetExample(GetExampleRequest) returns (GetExampleResponse) {
    option (google.api.http) = {
      get: "/api/v1/examples/{id}"
    };
    option (openapi.operation) = {
      summary: "获取示例"
      description: "根据ID获取示例信息"
      tags: "example"
    };
  }
}

message GetExampleRequest {
  string id = 1 [(validate.rules).string.min_len = 1];
}

message GetExampleResponse {
  string name = 1;
  int32 age = 2 [(validate.rules).int32.gte = 0];
  errors.Error error = 3;
}
` + "```" + `

## 生成代码

使用protoc命令生成代码时需要指定这些proto文件的路径：

` + "```bash" + `
protoc --proto_path=. \\
  --proto_path=./third_party \\
  --go_out=. \\
  --go-grpc_out=. \\
  --openapi_out=. \\
  your_proto_file.proto
` + "```" + `
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "README.md"), []byte(thirdPartyReadme), 0644); err != nil {
		return err
	}

	// 添加OpenAPI支持
	openapiAnnotationsProto := `syntax = "proto3";

package openapi;

option go_package = "github.com/dormoron/phantasm-openapi/annotations;annotations";

import "google/protobuf/descriptor.proto";

extend google.protobuf.MethodOptions {
  // OpenAPI操作元数据
  Operation operation = 1042;
}

extend google.protobuf.ServiceOptions {
  // OpenAPI信息元数据
  OpenAPI openapi = 1042;
}

extend google.protobuf.MessageOptions {
  // Schema元数据
  Schema schema = 1042;
}

extend google.protobuf.FieldOptions {
  // 字段元数据
  Field field = 1042;
}

// OpenAPI定义
message OpenAPI {
  // API标题
  string title = 1;
  // API描述
  string description = 2;
  // API版本
  string version = 3;
  // 联系信息
  Contact contact = 4;
}

// 联系信息
message Contact {
  // 联系人名称
  string name = 1;
  // 联系邮箱
  string email = 2;
  // 联系URL
  string url = 3;
}

// 操作定义
message Operation {
  // 操作摘要
  string summary = 1;
  // 操作详细描述
  string description = 2;
  // 操作标签，用于分组
  repeated string tags = 3;
  // 废弃标记
  bool deprecated = 4;
}

// Schema定义
message Schema {
  // 示例JSON
  string example = 1;
  // 描述
  string description = 2;
}

// 字段定义
message Field {
  // 描述
  string description = 1;
  // 示例值
  string example = 2;
  // 格式
  string format = 3;
  // 默认值
  string default = 4;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "openapi", "annotations.proto"), []byte(openapiAnnotationsProto), 0644); err != nil {
		return err
	}

	// 添加标准错误定义
	errorsProto := `syntax = "proto3";

package errors;

option go_package = "github.com/dormoron/phantasm/errors;errors";

// 统一错误响应
message Error {
  // 错误码
  int32 code = 1;
  // 错误消息
  string message = 2;
  // 错误详情
  repeated ErrorDetail details = 3;
}

// 错误详情
message ErrorDetail {
  // 错误类型
  string type = 1;
  // 错误字段
  string field = 2;
  // 详细消息
  string message = 3;
}

// 标准错误码定义
enum ErrorCode {
  // 成功
  OK = 0;
  
  // 客户端错误 (4xx)
  BAD_REQUEST = 400;
  UNAUTHORIZED = 401;
  FORBIDDEN = 403;
  NOT_FOUND = 404;
  METHOD_NOT_ALLOWED = 405;
  CONFLICT = 409;
  PRECONDITION_FAILED = 412;
  REQUEST_ENTITY_TOO_LARGE = 413;
  UNPROCESSABLE_ENTITY = 422;
  TOO_MANY_REQUESTS = 429;
  
  // 服务器错误 (5xx)
  INTERNAL_SERVER_ERROR = 500;
  NOT_IMPLEMENTED = 501;
  BAD_GATEWAY = 502;
  SERVICE_UNAVAILABLE = 503;
  GATEWAY_TIMEOUT = 504;
  
  // 自定义业务错误码 (1000+)
  BUSINESS_ERROR = 1000;
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "third_party", "errors", "errors.proto"), []byte(errorsProto), 0644); err != nil {
		return err
	}

	// 添加工具脚本
	makefileContent := `# 项目构建和管理工具

.PHONY: init
init: ## 初始化项目依赖
	go mod tidy

.PHONY: generate
generate: ## 生成代码
	go generate ./...

.PHONY: proto
proto: ## 生成proto文件
	protoc --proto_path=. \
		--proto_path=./third_party \
		--go_out=. \
		--go-grpc_out=. \
		./api/${APP_NAME}/v1/*.proto

.PHONY: build
build: ## 构建应用
	go build -o ./bin/ ./cmd/...

.PHONY: run
run: ## 运行应用
	go run ./cmd/${APP_NAME}/main.go -conf ./configs

.PHONY: test
test: ## 运行测试
	go test -v ./...

.PHONY: docker
docker: ## 构建Docker镜像
	docker build -t ${APP_NAME}:latest .

.PHONY: help
help: ## 显示帮助信息
	@echo "使用方法:"
	@echo " make [target]"
	@echo ""
	@echo "可用目标:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# 设置默认目标
.DEFAULT_GOAL := help
`
	if err := writeUTF8File(filepath.Join(projectPath, "Makefile"), []byte(makefileContent), 0644); err != nil {
		return err
	}

	// 生成脚本文件
	genProtoScript := `#!/bin/bash
# 用于生成proto文件的脚本

set -e

APP_NAME=$(basename $(pwd))
PROTO_FILES=$(find api -name "*.proto")

# 检查必要的工具
command -v protoc >/dev/null 2>&1 || { echo "错误: 需要安装protoc"; exit 1; }
command -v protoc-gen-go >/dev/null 2>&1 || { echo "错误: 需要安装protoc-gen-go"; exit 1; }
command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "错误: 需要安装protoc-gen-go-grpc"; exit 1; }

echo "开始生成Proto文件..."

for file in $PROTO_FILES; do
  echo "处理: $file"
  protoc --proto_path=. \
    --proto_path=./third_party \
    --go_out=. \
    --go-grpc_out=. \
    "$file"
done

echo "Proto生成完成"
`
	if err := writeUTF8File(filepath.Join(projectPath, "scripts", "gen_proto.sh"), []byte(genProtoScript), 0644); err != nil {
		return err
	}

	// 给脚本设置可执行权限
	if err := os.Chmod(filepath.Join(projectPath, "scripts", "gen_proto.sh"), 0755); err != nil {
		fmt.Printf("设置脚本可执行权限失败: %v\n", err)
	}

	// 创建内部错误处理包
	errorxCode := `package errorx

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error 表示应用错误
type Error struct {
	// 错误码
	Code int
	// 错误消息
	Message string
	// 错误详情
	Details []string
}

// New 创建新的错误
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WithDetails 添加错误详情
func (e *Error) WithDetails(details ...string) *Error {
	e.Details = append(e.Details, details...)
	return e
}

// Error 实现error接口
func (e *Error) Error() string {
	return fmt.Sprintf("错误: 代码=%d, 消息=%s, 详情=%v", e.Code, e.Message, e.Details)
}

// ToGRPCStatus 转换为gRPC状态
func (e *Error) ToGRPCStatus() *status.Status {
	return status.New(CodeToGRPCCode(e.Code), e.Message)
}

// CodeToGRPCCode 将错误码转换为gRPC代码
func CodeToGRPCCode(code int) codes.Code {
	switch code {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

// 预定义错误
var (
	// 客户端错误
	ErrBadRequest = New(http.StatusBadRequest, "无效的请求参数")
	ErrUnauthorized = New(http.StatusUnauthorized, "未授权")
	ErrForbidden = New(http.StatusForbidden, "禁止访问")
	ErrNotFound = New(http.StatusNotFound, "资源不存在")
	ErrTooManyRequests = New(http.StatusTooManyRequests, "请求过于频繁")
	
	// 服务器错误
	ErrInternalServer = New(http.StatusInternalServerError, "服务器内部错误")
	ErrServiceUnavailable = New(http.StatusServiceUnavailable, "服务不可用")
)
`
	if err := writeUTF8File(filepath.Join(projectPath, "internal", "pkg", "errorx", "error.go"), []byte(errorxCode), 0644); err != nil {
		return err
	}

	// 添加中间件文件
	loggerMiddleware := `package middleware

import (
	"github.com/dormoron/mist"
	"github.com/dormoron/phantasm/log"
	"time"
)

// Logger 日志中间件
func Logger(logger log.Logger) mist.HandlerFunc {
	return func(c *mist.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		
		// 处理请求
		c.Next()
		
		// 请求处理完成，记录日志
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		
		logger.Infof("| %3d | %13v | %15s | %s",
			statusCode,
			latency,
			c.ClientIP(),
			path,
		)
	}
}

// Recovery 恢复中间件
func Recovery(logger log.Logger) mist.HandlerFunc {
	return func(c *mist.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("请求处理时发生异常: %v", err)
				c.AbortWithStatus(500)
			}
		}()
		
		c.Next()
	}
}

// CORS 跨域中间件
func CORS() mist.HandlerFunc {
	return func(c *mist.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}
`
	if err := writeUTF8File(filepath.Join(projectPath, "internal", "pkg", "middleware", "middleware.go"), []byte(loggerMiddleware), 0644); err != nil {
		return err
	}

	// HTTP服务器
	if withHttp {
		httpServerContent := fmt.Sprintf(`package server

import (
    "github.com/dormoron/mist"
    "github.com/dormoron/phantasm/log"
    "github.com/dormoron/phantasm/middleware/logging"
    "github.com/dormoron/phantasm/middleware/recovery"
    "github.com/dormoron/phantasm/transport/http"
    
    "%s/internal/conf"
    "%s/internal/service"
)

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(c *conf.Server, logger log.Logger, svc *service.Service) *http.Server {
    var opts = []http.ServerOption{
        http.Address(c.Http.Addr),
        http.Timeout(c.Http.Timeout.AsDuration()),
        http.Logger(logger),
    }
    
    srv := http.NewServer(opts...)
    
    // 创建Mist引擎并设置中间件
    mServer, err := http.NewHTTPServer(
        http.WithAddress(c.Http.Addr),
        http.WithTimeout(c.Http.Timeout.AsDuration()),
    )
    if err != nil {
        panic(err)
    }
    
    // 使用中间件
    mServer.UseMiddleware(
        recovery.Recovery(),
        logging.Logging(
            logging.WithLogger(logger),
            logging.WithLogRequestBody(true),
            logging.WithLogResponseBody(true),
        ),
    )
    
    // 注册API路由组
    api := mServer.Group("/api")
    {
        v1 := api.Group("/v1")
        {
            v1.GET("/hello/:name", func(c *mist.Context) {
                nameVal, err := c.PathValue("name").String()
                if err != nil {
                    c.RespondWithJSON(400, map[string]string{"error": "无效的名称参数"})
                    return
                }
                // 调用服务实现
                message := "Hello " + nameVal
                c.RespondWithJSON(200, map[string]interface{}{
                    "message": message,
                })
            })
        }
    }
    
    // 健康检查
    mServer.GET("/health", func(c *mist.Context) {
        c.RespondWithJSON(200, map[string]string{"status": "ok"})
    })
    
    return srv
}`, module, module)

		if err := writeUTF8File(filepath.Join(projectPath, "internal", "server", "http.go"), []byte(httpServerContent), 0644); err != nil {
			return err
		}
	}

	// 更新API proto文件，添加OpenAPI注解
	apiProtoContent := fmt.Sprintf(`syntax = "proto3";

package api.%s.v1;

option go_package = "%s/api/%s/v1;v1";

import "google/api/annotations.proto";
import "validate/validate.proto";
import "openapi/annotations.proto";
import "errors/errors.proto";

service %s {
  option (openapi.openapi) = {
    title: "%s API"
    description: "基于Phantasm框架构建的微服务API"
    version: "v1.0.0"
    contact: {
      name: "开发团队"
      email: "team@example.com"
    }
  };

  rpc SayHello (HelloRequest) returns (HelloReply) {
    option (google.api.http) = {
      get: "/api/%s/hello/{name}"
    };
    option (openapi.operation) = {
      summary: "问候API"
      description: "返回一个带有名称的问候消息"
      tags: ["greeting"]
    };
  }
}

message HelloRequest {
  string name = 1 [
    (validate.rules).string = {min_len: 1, max_len: 100},
    (openapi.field) = {description: "要问候的名称", example: "世界"}
  ];
}

message HelloReply {
  string message = 1 [(openapi.field) = {description: "问候消息", example: "Hello 世界"}];
  errors.Error error = 2 [(openapi.field) = {description: "错误信息，成功时为null"}];
}
`, name, module, name, strings.Title(name), strings.Title(name), name)

	fmt.Println("创建API proto文件...")
	if err := writeUTF8File(filepath.Join(projectPath, "api", name, "v1", name+".proto"), []byte(apiProtoContent), 0644); err != nil {
		fmt.Printf("创建API proto文件失败: %v\n", err)
		return err
	}

	// 添加generate.go文件
	generateGo := fmt.Sprintf(`package main

//go:generate protoc --proto_path=. --proto_path=./third_party --go_out=. --go-grpc_out=. ./api/%s/v1/*.proto
`, name)
	if err := writeUTF8File(filepath.Join(projectPath, "cmd", name, "generate.go"), []byte(generateGo), 0644); err != nil {
		return err
	}

	// 添加.gitignore文件
	gitignoreContent := `# 编译生成的文件
/bin/
/dist/

# IDE和编辑器配置
.idea/
.vscode/
*.swp
*.swo
.DS_Store

# 依赖目录
/vendor/

# 日志文件
*.log

# 环境变量文件
.env

# 测试覆盖率文件
coverage.txt
profile.out

# 临时文件
tmp/
temp/

# 生成的配置文件
*.pb.go
`
	if err := writeUTF8File(filepath.Join(projectPath, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return err
	}

	// 添加OpenAPI生成脚本
	openapiGenScript := `#!/bin/bash
# 生成OpenAPI文档的脚本

set -e

APP_NAME=$(basename $(pwd))
PROTO_FILES=$(find api -name "*.proto")

# 检查必要工具
command -v protoc >/dev/null 2>&1 || { echo "错误: 需要安装protoc"; exit 1; }

echo "生成OpenAPI规范文档..."

mkdir -p docs/api

protoc --proto_path=. \
  --proto_path=./third_party \
  --openapiv2_out=docs/api \
  --openapiv2_opt=logtostderr=true \
  --openapiv2_opt=json_names_for_fields=true \
  $PROTO_FILES

echo "OpenAPI规范文档生成完成: docs/api/swagger.json"

# 检查是否有安装swagger-ui
if command -v swagger-ui >/dev/null 2>&1; then
  echo "通过swagger-ui查看API文档..."
  swagger-ui -p 8082 docs/api/swagger.json
else
  echo "如需查看API文档，请安装swagger-ui工具"
  echo "安装命令: npm install -g swagger-ui-cli"
fi
`
	if err := writeUTF8File(filepath.Join(projectPath, "scripts", "gen_openapi.sh"), []byte(openapiGenScript), 0644); err != nil {
		return err
	}

	// 给OpenAPI脚本设置可执行权限
	if err := os.Chmod(filepath.Join(projectPath, "scripts", "gen_openapi.sh"), 0755); err != nil {
		fmt.Printf("设置OpenAPI脚本可执行权限失败: %v\n", err)
	}

	// 输出成功信息
	fmt.Printf("\n🍺 项目创建成功 %s\n", color.GreenString(name))
	fmt.Print("💻 使用以下命令启动项目 👇:\n\n")

	fmt.Println(color.WhiteString("$ cd %s", name))
	fmt.Println(color.WhiteString("$ go mod tidy"))
	fmt.Println(color.WhiteString("$ go generate ./..."))
	fmt.Println(color.WhiteString("$ go build -o ./bin/ ./... "))
	fmt.Println(color.WhiteString("$ ./bin/%s -conf ./configs\n", name))
	fmt.Println("			🤝 感谢使用Phantasm")

	// 创建README.md
	readmeContent := fmt.Sprintf(`# %s

基于Phantasm框架构建的微服务项目

## 介绍

这是一个使用Phantasm框架创建的微服务项目，集成了mist作为Web框架和eidola作为gRPC框架。

## 特性

- 完整的微服务架构
- HTTP与gRPC协议支持
- 中间件支持（日志、恢复、CORS等）
- 统一错误处理
- 参数验证
- OpenAPI规范支持

## 目录结构

- **api/**: API定义 (Protocol Buffers)
- **cmd/**: 应用程序入口
- **configs/**: 配置文件
- **internal/**: 内部代码
  - **biz/**: 业务逻辑层
  - **data/**: 数据访问层
  - **server/**: 服务器初始化
  - **service/**: 服务实现
  - **conf/**: 配置结构定义
  - **pkg/**: 内部共享包
    - **middleware/**: HTTP中间件
    - **errorx/**: 错误处理
- **scripts/**: 工具脚本
- **third_party/**: 第三方proto文件

## 快速开始

### 安装依赖

`+"```bash"+`
make init
`+"```"+`

### 生成代码

`+"```bash"+`
# 生成proto相关代码
make proto
# 或者
./scripts/gen_proto.sh

# 生成所有代码
make generate
`+"```"+`

### 运行

`+"```bash"+`
# 使用make运行
make run

# 或者直接运行
go run ./cmd/%s/main.go -conf ./configs
`+"```"+`

### 构建

`+"```bash"+`
make build
`+"```"+`

### 生成API文档

`+"```bash"+`
./scripts/gen_openapi.sh
`+"```"+`

## Docker支持

`+"```bash"+`
# 构建Docker镜像
make docker

# 运行Docker容器
docker run -p 8000:8000 -p 9000:9000 %s:latest
`+"```"+`

## 配置

配置文件位于 configs/config.yaml，支持以下配置：

- HTTP服务器配置 (地址、超时)
- gRPC服务器配置 (地址、超时)
- 数据库配置
- Redis配置

## 项目导航

- HTTP服务: http://localhost:8000
- gRPC服务: localhost:9000
- API文档: http://localhost:8082 (运行gen_openapi.sh后)
- 健康检查: http://localhost:8000/health

## 帮助

查看所有可用的make命令：

`+"```bash"+`
make help
`+"```"+`
`,
		name,
		name,
		name)

	if err := writeUTF8File(filepath.Join(projectPath, "README.md"), []byte(readmeContent), 0644); err != nil {
		return err
	}

	return nil
}

// 添加辅助函数
func getHttpImport(withHttp bool) string {
	if withHttp {
		return `"github.com/dormoron/phantasm/transport/http"`
	}
	return ""
}

func getGrpcImport(withGrpc bool) string {
	if withGrpc {
		return `"github.com/dormoron/phantasm/transport/grpc"`
	}
	return ""
}

func getHttpServer(withHttp bool) string {
	if withHttp {
		return `phantasm.Server(newHTTPServer(zlog)),`
	}
	return ""
}

func getGrpcServer(withGrpc bool) string {
	if withGrpc {
		return `phantasm.Server(newGRPCServer(zlog)),`
	}
	return ""
}

// 获取服务器初始化代码
func getServerInit(enabled bool, code string) string {
	if enabled {
		return code
	}
	return ""
}

// writeUTF8File 将内容以UTF-8编码写入文件，并在Windows系统上添加BOM
func writeUTF8File(filePath string, content []byte, perm os.FileMode) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("无法创建目录 %s: %v\n", dir, err)
		return err
	}

	// 这些文件类型不应添加 BOM，否则会导致工具链报错或解析问题
	ext := filepath.Ext(filePath)
	fileName := filepath.Base(filePath)
	skipBOM := false

	// 跳过 BOM 的文件类型列表
	if fileName == "go.mod" || fileName == "go.sum" ||
		ext == ".go" || ext == ".sh" || ext == ".yaml" || ext == ".yml" ||
		ext == ".json" || ext == ".proto" || ext == ".mod" || ext == ".sum" {
		skipBOM = true
	}

	if skipBOM {
		// 直接写入内容，不添加 BOM
		err := os.WriteFile(filePath, content, perm)
		if err != nil {
			fmt.Printf("写入文件 %s 失败: %v\n", filePath, err)
			return err
		}
		fmt.Printf("成功写入文件: %s (%d 字节)\n", filePath, len(content))
		return nil
	}

	// 添加UTF-8 BOM (Byte Order Mark)，确保Windows系统正确识别UTF-8编码
	// BOM是可选的，但在Windows中有助于确保正确识别文件编码
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}

	// 判断内容是否已经有BOM
	hasUTF8BOM := false
	if len(content) >= 3 {
		hasUTF8BOM = content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF
	}

	// 如果没有BOM则添加
	var finalContent []byte
	if !hasUTF8BOM {
		finalContent = append(utf8BOM, content...)
	} else {
		finalContent = content
	}

	// 写入文件
	err := os.WriteFile(filePath, finalContent, perm)
	if err != nil {
		fmt.Printf("写入文件 %s 失败: %v\n", filePath, err)
		return err
	}

	fmt.Printf("成功写入文件: %s (%d 字节)\n", filePath, len(finalContent))
	return nil
}
