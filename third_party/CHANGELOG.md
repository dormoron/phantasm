# Third Party 变更日志

## v1.0.0 (2024-03-25)

### 新增功能

- 添加 Google API 相关 proto 文件
  - `google/api/annotations.proto`: HTTP 与 gRPC 映射注解
  - `google/api/http.proto`: HTTP 规则定义
  - `google/api/httpbody.proto`: HTTP Body 定义
  
- 添加参数验证相关 proto 文件
  - `validate/validate.proto`: 验证规则定义，基于 envoyproxy/protoc-gen-validate
  
- 添加 OpenAPI/Swagger 相关 proto 文件
  - `openapi/annotations.proto`: OpenAPI 注解定义

### 文档更新

- 为 third_party 目录添加总体 README.md
- 为 validate 目录添加专门的 README.md
- 为 openapi 目录添加专门的 README.md
- 在 docs/guide 目录下添加 third-party.md 使用指南

### 工具支持

- 添加 tools/protoc.sh 脚本，用于从 proto 文件生成 Go 代码、gRPC 代码和 Swagger 文档

### 示例

- 添加 api/example/v1/example.proto 作为示例，展示如何使用 third_party 中的各种 proto 文件 