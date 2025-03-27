# Phantasm: Elegant and Powerful Go Microservices Framework

Phantasm is an elegant and powerful Go microservices framework designed for building high-performance, highly reliable microservice applications. The framework integrates `mist` as the web framework and `eidola` as the gRPC framework, providing a complete solution from development to deployment.

English | [中文](./README.md)

## 🌟 Core Features

- **Simple API Design**: Intuitive and easy-to-use API interfaces, reducing learning costs and improving development efficiency
- **Modular Architecture**: Loosely coupled component design, supporting flexible assembly and customized development
- **Multi-protocol Support**: Simultaneously supporting HTTP and gRPC to meet communication needs in different scenarios
- **Service Governance**: Built-in service registration and discovery, load balancing, health checks, and more
- **Full-link Tracing**: Integration with OpenTelemetry, supporting distributed tracing and metrics collection
- **Structured Logging**: High-performance logging system based on zap, supporting multiple levels and outputs
- **Rich Middleware**: Provides authentication, authorization, rate limiting, circuit breaking, and other common middleware
- **Configuration Management**: Flexible configuration loading mechanism, supporting multiple configuration sources and dynamic updates
- **Unified Error Handling**: Standardized error definitions and handling mechanisms, improving API consistency
- **Buf Integration**: Built-in Buf tool support, enhancing Proto file management and validation capabilities
- **Graceful Startup/Shutdown**: Support for smooth startup and shutdown, ensuring service reliability
- **Development Toolchain**: Provides command-line tools to simplify project creation and code generation

## 📦 Installation

### Prerequisites

- Go 1.23 or higher
- Protoc installation required for protobuf functionality
- Optional installation of buf tool to enhance Proto file management experience

### Installing the CLI Tool

```bash
go install github.com/dormoron/phantasm/cmd/phantasm@latest
```

### Adding Framework Dependencies

```bash
go get github.com/dormoron/phantasm@latest
```

## 🚀 Quick Start

### Creating a New Project

Use the CLI tool to quickly create a project:

```bash
# Create a new project named "helloworld"
phantasm new helloworld

# Enter the project directory
cd helloworld
```

### Project Structure

The generated project structure is as follows:

```
.
├── api                # API definitions directory (protobuf files)
├── cmd                # Command-line entry points
│   └── server         # Service startup entry
├── configs            # Configuration files directory
├── internal           # Internal code packages
│   ├── biz            # Business logic layer
│   ├── data           # Data access layer
│   ├── server         # Service implementation layer
│   └── service        # Service interface layer
├── third_party        # Third-party dependencies
│   ├── errors         # Standard error definitions
│   ├── buf            # Buf-related configurations
│   └── google         # Google API definitions
├── buf.yaml           # Buf tool configuration
├── go.mod             # Go module definition
├── go.sum             # Go dependency lock file
└── README.md          # Project documentation
```

### Starting the Service

```bash
# Run directly
go run ./cmd/server

# Or use the Phantasm tool to run
phantasm run
```

## 💡 Core Concepts and Usage

### Application Lifecycle

Phantasm provides complete application lifecycle management, including preparation before startup, startup, running, and shutdown phases:

```go
app := phantasm.New(
    phantasm.Name("my-service"),        // Service name
    phantasm.Version("v1.0.0"),         // Service version
    phantasm.Server(httpServer, grpcServer), // Register servers
    phantasm.BeforeStart(func(ctx context.Context) error {
        // Preparation work before startup
        return nil
    }),
    phantasm.AfterStart(func(ctx context.Context) error {
        // Operations after successful startup
        return nil
    }),
    phantasm.BeforeStop(func(ctx context.Context) error {
        // Cleanup work before stopping
        return nil
    }),
    phantasm.AfterStop(func(ctx context.Context) error {
        // Operations after stopping
        return nil
    }),
)

// Start the application
if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

### HTTP Service

HTTP service support based on the `mist` framework:

```go
import (
    "phantasm/transport/http"
    "github.com/dormoron/mist"
)

// Create HTTP server
httpServer, err := http.NewHTTPServer(
    http.WithAddress(":8000"),
    http.WithTimeout(time.Second*5),
    http.WithMiddleware(
        middleware.Recovery(),
        middleware.Logger(),
    ),
)
if err != nil {
    log.Fatal(err)
}

// Register routes
httpServer.GET("/hello", func(c *mist.Context) {
    c.RespondWithJSON(200, map[string]interface{}{
        "message": "Hello, Phantasm!",
    })
})

// Register service and start
app := phantasm.New(
    phantasm.Name("http-service"),
    phantasm.Server(httpServer),
)

if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

### gRPC Service

gRPC service support based on the `eidola` framework:

```go
import (
    "phantasm/transport/grpc"
    helloworldpb "myapp/api/helloworld/v1"
)

// Create gRPC server
grpcServer, err := grpc.NewGRPCServer(
    grpc.WithAddress(":9000"),
    grpc.WithTimeout(time.Second*5),
    grpc.WithMiddleware(
        grpcmiddleware.Recovery(),
        grpcmiddleware.Logger(),
    ),
)
if err != nil {
    log.Fatal(err)
}

// Register service
helloworldpb.RegisterGreeterServer(grpcServer.Server(), &GreeterService{})

// Register service and start
app := phantasm.New(
    phantasm.Name("grpc-service"),
    phantasm.Server(grpcServer),
)

if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

### Service Registration and Discovery

Support for multiple registry centers, including Consul, Etcd, Zookeeper, etc.:

```go
import (
    "phantasm/registry"
    "phantasm/registry/consul"
)

// Create Consul registry center
r, err := consul.New(consul.WithAddress("127.0.0.1:8500"))
if err != nil {
    log.Fatal(err)
}

// Use in application
app := phantasm.New(
    phantasm.Name("my-service"),
    phantasm.Version("v1.0.0"),
    phantasm.Server(httpServer, grpcServer),
    phantasm.Registrar(r),
)
```

### Configuration Management

Based on a flexible configuration management system:

```go
import (
    "phantasm/config"
    "phantasm/config/file"
)

// Load configuration files
c, err := config.Load(
    file.NewSource("configs/config.yaml"),
    file.NewSource("configs/config.local.yaml", file.WithOptional(true)),
)
if err != nil {
    log.Fatal(err)
}

// Get configuration values
httpPort := c.Get("server.http.port").String("8000")
```

### Logging System

High-performance structured logging system:

```go
import (
    "phantasm/log"
)

// Create logger
logger := log.NewLogger()

// Record logs of different levels
logger.Debug("Debug information", log.String("key", "value"))
logger.Info("General information", log.Int("count", 100))
logger.Warn("Warning information", log.Duration("elapsed", time.Second))
logger.Error("Error information", log.Err(err))
```

### Error Handling

Phantasm provides a standardized error handling mechanism to keep API error responses consistent:

```go
import (
    "github.com/dormoron/phantasm/errors"
)

// Create a new error
err := errors.New(
    errors.CodeNotFound,    // Error code
    "user.not_found",       // Error reason
    "User does not exist",  // Error message
)

// Add metadata
err = err.WithMetadata(map[string]string{
    "user_id": "123456",
})

// Use in HTTP/gRPC responses
if user == nil {
    return nil, err
}

// Extract information from errors
if e, ok := errors.FromError(err); ok {
    code := e.Code      // Get error code
    reason := e.Reason  // Get error reason
    message := e.Message // Get error message
}
```

### Buf Tool Integration

Phantasm has built-in support for the Buf tool, providing a better Proto file management experience:

```bash
# Initialize Buf workspace
phantasm proto init

# Validate Proto files with buf
phantasm proto lint

# Check breaking changes with buf
phantasm proto breaking

# Generate code with buf
phantasm proto generate
```

You can also run native buf commands using the configuration provided in the project:

```bash
buf lint
buf build
buf generate
```

## 🔧 Toolchain

### Project Creation

```bash
# Create standard project
phantasm new myproject

# Create HTTP-only project
phantasm new myproject --http-only

# Create gRPC-only project
phantasm new myproject --grpc-only
```

### Protobuf Code Generation

```bash
# Generate protobuf code
phantasm proto generate

# Generate HTTP gateway
phantasm proto generate --with-http-gateway
```

### Running Projects

```bash
# Run project
phantasm run

# Run in specific environment
phantasm run --env=production
```

### Upgrading Framework

```bash
# Upgrade framework to latest version
phantasm upgrade
```

## 📚 Project Structure

```
├── api/                 # API definitions (Protobuf)
├── cmd/                 # Command-line tools
│   ├── phantasm/          # CLI tool
│   └── protoc-gen-phantasm-http/ # Protobuf code generator
├── config/              # Configuration management
├── contrib/             # Third-party integrations
├── encoding/            # Encoders
├── errors/              # Error handling
├── examples/            # Example code
├── internal/            # Internal packages
├── log/                 # Logging
├── metadata/            # Metadata
├── middleware/          # Middleware
├── registry/            # Registry center
├── selector/            # Load balancing
├── tools/               # Development tools
├── transport/           # Transport layer
│   ├── grpc/            # gRPC transport
│   └── http/            # HTTP transport
├── app.go               # Application entry
├── options.go           # Option definitions
└── version.go           # Version information
```

## 📊 Performance Benchmarks

In a standard test environment (8-core CPU, 16GB memory), the performance of the Phantasm framework:

- HTTP service: ~50,000 QPS
- gRPC service: ~100,000 QPS
- Memory usage: ~20MB (basic startup)
- Startup time: <500ms

## 🔍 Comparison with Other Frameworks

| Feature | Phantasm | Go-Kit | Go-Micro | Kratos |
|---------|---------|--------|----------|--------|
| Ease of use | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Performance | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| Feature completeness | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Community support | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Documentation quality | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |

## 🤝 Contribution Guidelines

We warmly welcome community contributions, whether reporting issues, suggesting improvements, or submitting code - all help make Phantasm better!

For detailed contribution guidelines, please refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 License

The Phantasm framework is open-source under the [MIT License](LICENSE).

## 🔗 Related Projects

- [mist](https://github.com/dormoron/mist) - Lightweight HTTP framework
- [eidola](https://github.com/dormoron/eidola) - Powerful gRPC framework
- [phantasm-examples](https://github.com/dormoron/Phantasm-examples) - Collection of example projects

## 📮 Contact Us

- GitHub Issues: [https://github.com/dormoron/phantasm/issues](https://github.com/dormoron/Phantasm/issues)
- Email: dormoron@163.com.com
- Community: [https://discord.gg/phantasm-framework](https://discord.gg/Phantasm-framework)

## 🔧 More Examples

### Using Resource Definitions

Leverage resource definitions provided by Google API to enhance API expressiveness:

```protobuf
syntax = "proto3";

package api.example.v1;

import "google/api/resource.proto";
import "google/api/field_behavior.proto";

message User {
  option (google.api.resource) = {
    type: "example.com/User"
    pattern: "users/{user_id}"
    name_field: "name"
  };

  // Resource name
  string name = 1 [(google.api.field_behavior) = IDENTIFIER];
  
  // Username
  string username = 2 [(google.api.field_behavior) = REQUIRED];
}
```

### Using Standard Errors

Use standard error responses in APIs:

```protobuf
syntax = "proto3";

package api.example.v1;

import "third_party/errors/errors.proto";

message GetUserResponse {
  // User data
  User user = 1;
  
  // Standard error response
  errors.ErrorResponse error = 2;
}
```

## 📚 Documentation

For complete documentation, visit [https://phantasm.dev](https://phantasm.dev)

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## 📄 License

Phantasm is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details. 