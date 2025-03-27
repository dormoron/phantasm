# Protocol Buffer Validation

这个目录包含用于协议缓冲区验证的Proto定义，基于 [envoyproxy/protoc-gen-validate](https://github.com/envoyproxy/protoc-gen-validate) 项目。

## 使用方法

在你的proto文件中，导入validate.proto并在字段上应用规则：

```protobuf
import "validate/validate.proto";

message Person {
  // name字段必须非空且长度在1到64个字符之间
  string name = 1 [(validate.rules).string = {min_len: 1, max_len: 64}];
  
  // age字段必须是在0到120之间的数字
  int32 age = 2 [(validate.rules).int32 = {gte: 0, lte: 120}];
  
  // email字段必须是有效的电子邮件格式
  string email = 3 [(validate.rules).string.email = true];
}
```

## 生成验证代码

使用以下命令生成验证代码：

```bash
protoc --proto_path=. \
       --proto_path=./third_party \
       --go_out=paths=source_relative:. \
       --validate_out=paths=source_relative,lang=go:. \
       path/to/your/file.proto
```

## 中间件集成

Cosmos框架提供了验证中间件，可以自动验证请求参数：

```go
// HTTP服务器
httpSrv := http.NewServer(
    http.Address(":8000"),
    http.Middleware(
        validate.Validator(),
    ))

// gRPC服务器
grpcSrv := grpc.NewServer(
    grpc.Address(":9000"),
    grpc.Middleware(
        validate.Validator(),
    ))
```

## 参考

- [protoc-gen-validate](https://github.com/envoyproxy/protoc-gen-validate)
- [Validation Rules](https://github.com/envoyproxy/protoc-gen-validate#constraint-rules) 