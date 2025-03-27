# OpenAPI (Swagger) 支持

该目录包含用于生成OpenAPI/Swagger规范的Proto定义文件。这些文件允许您在Proto文件中添加OpenAPI注解，以便可以从Proto文件生成Swagger文档。

## 使用

要在您的API中使用OpenAPI/Swagger支持，您需要：

1. 在您的proto文件中导入此目录下的proto文件
2. 使用相关的注解来定义OpenAPI/Swagger文档
3. 使用支持的工具生成OpenAPI/Swagger规范文件

### 示例

```protobuf
syntax = "proto3";

package example.v1;

import "google/api/annotations.proto";
import "openapi/annotations.proto";

option go_package = "example/api/v1;v1";

service ExampleService {
  option (openapi.tag) = {
    name: "Example"
    description: "示例服务"
  };

  rpc GetExample(GetExampleRequest) returns (GetExampleResponse) {
    option (google.api.http) = {
      get: "/v1/examples/{id}"
    };
    option (openapi.operation) = {
      summary: "获取示例"
      description: "获取指定ID的示例信息"
      tags: ["Example"]
    };
  }
}

message GetExampleRequest {
  string id = 1 [(openapi.property) = {
    description: "示例ID"
    min_length: 1
  }];
}

message GetExampleResponse {
  string id = 1;
  string name = 2;
  string description = 3;
}
```

## 生成Swagger文档

使用protoc和相关插件来生成swagger.json文件：

```bash
protoc --proto_path=. \
       --proto_path=./third_party \
       --openapi_out=./api/swagger \
       your_service.proto
```

## 参考

- [OpenAPI规范](https://github.com/OAI/OpenAPI-Specification)
- [Swagger](https://swagger.io/)
- [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway) 