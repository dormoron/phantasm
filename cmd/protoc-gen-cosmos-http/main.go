package main

import (
	"flag"
	"fmt"
	"github.com/dormoron/phantasm"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	showVersion := flag.Bool("version", false, "打印版本号")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-phantasm-http %s\n", phantasm.VERSION)
		return
	}

	protogen.Options{
		ParamFunc: flag.CommandLine.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f)
		}
		return nil
	})
}

// generateFile 为单个.proto文件生成HTTP处理器
func generateFile(gen *protogen.Plugin, file *protogen.File) {
	if len(file.Services) == 0 {
		return
	}
	filename := file.GeneratedFilenamePrefix + "_http.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	g.P("// Code generated by protoc-gen-phantasm-http. DO NOT EDIT.")
	g.P("// versions:")
	g.P("// protoc-gen-phantasm-http ", phantasm.VERSION)
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()

	g.P("import (")
	g.P(`	"context"`)
	g.P(`	"net/http"`)
	g.P()
	g.P(`	"github.com/dormoron/mist"`)
	g.P(`	"google.golang.org/protobuf/encoding/protojson"`)
	g.P(`)`)
	g.P()

	// 为每个服务生成HTTP处理器
	for _, service := range file.Services {
		generateHTTPService(gen, file, g, service)
	}
}

// generateHTTPService 生成HTTP服务处理器
func generateHTTPService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serviceName := service.GoName

	// 定义HTTP服务接口
	g.P("// ", serviceName, "HTTPServer 是", serviceName, "的HTTP服务器接口")
	g.P("type ", serviceName, "HTTPServer interface {")
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue // 跳过流式方法
		}
		g.P("	", method.GoName, "(context.Context, *", method.Input.GoIdent, ") (*", method.Output.GoIdent, ", error)")
	}
	g.P("}")
	g.P()

	// 定义HTTP服务器
	g.P("// Register", serviceName, "HTTPServer 将服务处理程序注册到HTTP路由器")
	g.P("func Register", serviceName, "HTTPServer(r *mist.HTTPServer, srv ", serviceName, "HTTPServer) {")
	g.P("	h := new", serviceName, "Handler(srv)")
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue // 跳过流式方法
		}
		path := fmt.Sprintf("/%s/%s", service.Desc.Name(), method.Desc.Name())
		g.P(`	r.POST("`, path, `", h.`, method.GoName, ")")
	}
	g.P("}")
	g.P()

	// 定义处理器结构体
	g.P("type ", unexport(serviceName), "Handler struct {")
	g.P("	srv ", serviceName, "HTTPServer")
	g.P("}")
	g.P()

	// 定义创建处理器函数
	g.P("func new", serviceName, "Handler(srv ", serviceName, "HTTPServer) *", unexport(serviceName), "Handler {")
	g.P("	return &", unexport(serviceName), "Handler{")
	g.P("		srv: srv,")
	g.P("	}")
	g.P("}")
	g.P()

	// 为每个方法定义HTTP处理函数
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue // 跳过流式方法
		}
		g.P("func (h *", unexport(serviceName), "Handler) ", method.GoName, "(c *mist.Context) {")
		g.P("	var req ", method.Input.GoIdent)
		g.P("	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {")
		g.P("		c.RespondWithJSON(http.StatusBadRequest, map[string]interface{}{")
		g.P(`			"error": err.Error(),`)
		g.P("		})")
		g.P("		return")
		g.P("	}")
		g.P("	resp, err := h.srv.", method.GoName, "(c.Request.Context(), &req)")
		g.P("	if err != nil {")
		g.P("		c.RespondWithJSON(http.StatusInternalServerError, map[string]interface{}{")
		g.P(`			"error": err.Error(),`)
		g.P("		})")
		g.P("		return")
		g.P("	}")
		g.P("	c.RespondWithJSON(http.StatusOK, resp)")
		g.P("}")
		g.P()
	}

	// 生成客户端代码
	generateHTTPClient(gen, file, g, service)
}

// generateHTTPClient 生成HTTP客户端
func generateHTTPClient(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	serviceName := service.GoName

	// 导入所需包
	g.P("import (")
	g.P(`	"bytes"`)
	g.P(`	"context"`)
	g.P(`	"encoding/json"`)
	g.P(`	"fmt"`)
	g.P(`	"io"`)
	g.P(`	"net/http"`)
	g.P(")")
	g.P()

	// 定义客户端接口
	g.P("// ", serviceName, "HTTPClient 是", serviceName, "的HTTP客户端接口")
	g.P("type ", serviceName, "HTTPClient interface {")
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue // 跳过流式方法
		}
		g.P("	", method.GoName, "(ctx context.Context, req *", method.Input.GoIdent, ") (*", method.Output.GoIdent, ", error)")
	}
	g.P("}")
	g.P()

	// 定义HTTP客户端结构体
	g.P("type ", unexport(serviceName), "HTTPClient struct {")
	g.P("	client *http.Client")
	g.P("	baseURL string")
	g.P("}")
	g.P()

	// 定义创建客户端函数
	g.P("// New", serviceName, "HTTPClient 创建一个新的HTTP客户端")
	g.P("func New", serviceName, "HTTPClient(baseURL string) ", serviceName, "HTTPClient {")
	g.P("	return &", unexport(serviceName), "HTTPClient{")
	g.P("		client: http.DefaultClient,")
	g.P("		baseURL: baseURL,")
	g.P("	}")
	g.P("}")
	g.P()

	// 为每个方法定义客户端方法
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue // 跳过流式方法
		}
		g.P("func (c *", unexport(serviceName), "HTTPClient) ", method.GoName, "(ctx context.Context, req *", method.Input.GoIdent, ") (*", method.Output.GoIdent, ", error) {")
		g.P("	path := fmt.Sprintf(\"%s/%s/%s\", c.baseURL, \"", service.Desc.Name(), "\", \"", method.Desc.Name(), "\")")
		g.P("	data, err := json.Marshal(req)")
		g.P("	if err != nil {")
		g.P("		return nil, err")
		g.P("	}")
		g.P("	httpReq, err := http.NewRequestWithContext(ctx, \"POST\", path, bytes.NewReader(data))")
		g.P("	if err != nil {")
		g.P("		return nil, err")
		g.P("	}")
		g.P(`	httpReq.Header.Set("Content-Type", "application/json")`)
		g.P("	resp, err := c.client.Do(httpReq)")
		g.P("	if err != nil {")
		g.P("		return nil, err")
		g.P("	}")
		g.P("	defer resp.Body.Close()")
		g.P("	body, err := io.ReadAll(resp.Body)")
		g.P("	if err != nil {")
		g.P("		return nil, err")
		g.P("	}")
		g.P("	if resp.StatusCode != http.StatusOK {")
		g.P("		return nil, fmt.Errorf(\"unexpected status code: %d, body: %s\", resp.StatusCode, string(body))")
		g.P("	}")
		g.P("	var result ", method.Output.GoIdent)
		g.P("	if err := json.Unmarshal(body, &result); err != nil {")
		g.P("		return nil, err")
		g.P("	}")
		g.P("	return &result, nil")
		g.P("}")
		g.P()
	}
}

// unexport 将首字母小写
func unexport(s string) string {
	if len(s) == 0 {
		return ""
	}
	r := []rune(s)
	r[0] = toLower(r[0])
	return string(r)
}

// toLower 将单个字符转换为小写
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
