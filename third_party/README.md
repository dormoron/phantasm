# Third Party Proto Files

此目录包含第三方的Proto文件定义，用于支持Cosmos框架的API定义功能。

## 内容

- `google/api/`: Google API相关的proto文件，用于HTTP与gRPC的映射
  - `annotations.proto`: 基本的HTTP注解定义
  - `http.proto`: HTTP规则和路径定义
  - `httpbody.proto`: HTTP响应体定义
  - `field_behavior.proto`: 字段行为标记（必填、只读等）
  - `resource.proto`: 资源定义和资源引用
  - `client.proto`: 客户端选项定义
- `validate/`: 参数校验相关的proto文件，基于envoyproxy/protoc-gen-validate
- `openapi/`: OpenAPI (Swagger) 相关的proto文件

## 使用方法

在定义API时，可以引入这些proto文件，例如：

```protobuf
import "google/api/annotations.proto";
import "google/api/field_behavior.proto";
import "google/api/resource.proto";
import "validate/validate.proto";

// 定义资源
message Resource {
  option (google.api.resource) = {
    type: "example.com/Resource"
    pattern: "resources/{resource}"
  };

  // 资源ID
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference).type = "example.com/Resource"
  ];
  
  // 用户提供的名称
  string display_name = 2 [(validate.rules).string.min_len = 1];
  
  // 创建时间 (只读)
  int64 create_time = 3 [(google.api.field_behavior) = OUTPUT_ONLY];
}

// 定义服务
service ResourceService {
  rpc GetResource(GetResourceRequest) returns (Resource) {
    option (google.api.http) = {
      get: "/v1/{name=resources/*}"
    };
  }
  
  rpc CreateResource(CreateResourceRequest) returns (Resource) {
    option (google.api.http) = {
      post: "/v1/resources"
      body: "resource"
    };
  }
}

// 获取资源请求
message GetResourceRequest {
  // 资源名称
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference).type = "example.com/Resource"
  ];
}

// 创建资源请求
message CreateResourceRequest {
  // 要创建的资源
  Resource resource = 1 [(google.api.field_behavior) = REQUIRED];
}
```

## 更新

这些文件来自以下仓库：

- Google API: https://github.com/googleapis/googleapis
- Validate: https://github.com/envoyproxy/protoc-gen-validate
- OpenAPI: https://github.com/google/gnostic