@echo off
REM Windows batch script for generating protobuf code

REM 设置控制台代码页为UTF-8
chcp 65001
echo 设置控制台为UTF-8编码

setlocal enabledelayedexpansion

REM 项目根目录
set ROOT_DIR=%~dp0..
REM API目录
set API_DIR=%ROOT_DIR%\api
REM API文档输出目录
set SWAGGER_OUT_DIR=%ROOT_DIR%\api\swagger
REM Google API目录
set GOOGLE_API_DIR=%ROOT_DIR%\third_party\google\api
REM Errors目录
set ERRORS_DIR=%ROOT_DIR%\third_party\errors
REM Buf目录
set BUF_DIR=%ROOT_DIR%\third_party\buf

REM 确保输出目录存在
if not exist "%SWAGGER_OUT_DIR%" mkdir "%SWAGGER_OUT_DIR%"

REM 检查是否安装了所需的插件
where protoc >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Error: protoc is not installed. Please install it first.
    exit /b 1
)

where protoc-gen-go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Error: protoc-gen-go is not installed. Please install it using:
    echo go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    exit /b 1
)

where protoc-gen-go-grpc >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Error: protoc-gen-go-grpc is not installed. Please install it using:
    echo go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    exit /b 1
)

where protoc-gen-validate >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Error: protoc-gen-validate is not installed. Please install it using:
    echo go install github.com/envoyproxy/protoc-gen-validate@latest
    exit /b 1
)

where protoc-gen-openapiv2 >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Error: protoc-gen-openapiv2 is not installed. Please install it using:
    echo go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
    exit /b 1
)

REM 检查buf是否安装
where buf >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Warning: buf is not installed. Will use protoc directly. 
    echo To install buf run: go install github.com/bufbuild/buf/cmd/buf@latest
)

echo ===== 第1步：生成Google API依赖的Proto文件 =====
echo 处理Google API目录: %GOOGLE_API_DIR%

REM 生成Google API基础文件
cd %ROOT_DIR%
protoc --proto_path=third_party ^
       --go_out=. ^
       third_party/google/api/http.proto ^
       third_party/google/api/annotations.proto ^
       third_party/google/api/httpbody.proto

REM 生成Google API扩展文件（如果存在）
if exist "third_party\google\api\field_behavior.proto" (
    echo 生成field_behavior.proto
    protoc --proto_path=third_party ^
           --go_out=. ^
           third_party/google/api/field_behavior.proto
)

if exist "third_party\google\api\resource.proto" (
    echo 生成resource.proto
    protoc --proto_path=third_party ^
           --go_out=. ^
           third_party/google/api/resource.proto
)

if exist "third_party\google\api\client.proto" (
    echo 生成client.proto
    protoc --proto_path=third_party ^
           --go_out=. ^
           third_party/google/api/client.proto
)

echo ===== 第2步：生成Errors和Buf依赖的Proto文件 =====
if exist "third_party\errors\errors.proto" (
    echo 处理文件: third_party\errors\errors.proto
    
    REM 生成Go代码
    protoc --proto_path=third_party ^
           --go_out=. ^
           third_party/errors/errors.proto
)

if exist "third_party\buf\buf.proto" (
    echo 处理文件: third_party\buf\buf.proto
    
    REM 生成Go代码
    protoc --proto_path=third_party ^
           --go_out=. ^
           third_party/buf/buf.proto
)

echo ===== 第3步：生成资源示例的API文件 =====
if exist "api\example\v1\resource_example.proto" (
    echo 处理文件: api\example\v1\resource_example.proto
    
    REM 生成Go代码
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --go_out=Mapi/example/v1/resource_example.proto=github.com/dormoron/phantasm/api/example/v1:. ^
           api/example/v1/resource_example.proto
    
    echo 生成gRPC代码...
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --go-grpc_out=Mapi/example/v1/resource_example.proto=github.com/dormoron/phantasm/api/example/v1:. ^
           api/example/v1/resource_example.proto
    
    echo 生成验证代码...
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --validate_out="lang=go:." ^
           api/example/v1/resource_example.proto
)

echo ===== 第4步：生成基本示例的API文件 =====
if exist "api\example\v1\example.proto" (
    echo 处理文件: api\example\v1\example.proto
    
    REM 生成Go代码
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --go_out=. ^
           api/example/v1/example.proto
    
    echo 生成gRPC代码...
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --go-grpc_out=. ^
           api/example/v1/example.proto
    
    echo 生成验证代码...
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --validate_out="lang=go:." ^
           api/example/v1/example.proto
)

echo ===== 第5步：生成错误处理示例的API文件 =====
if exist "api\example\v1\errors_example.proto" (
    echo 处理文件: api\example\v1\errors_example.proto
    
    REM 生成Go代码
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --go_out=Mapi/example/v1/errors_example.proto=github.com/dormoron/phantasm/api/example/v1:. ^
           api/example/v1/errors_example.proto
    
    echo 生成gRPC代码...
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --go-grpc_out=Mapi/example/v1/errors_example.proto=github.com/dormoron/phantasm/api/example/v1:. ^
           api/example/v1/errors_example.proto
    
    echo 生成验证代码...
    protoc --proto_path=third_party ^
           --proto_path=. ^
           --validate_out="lang=go:." ^
           api/example/v1/errors_example.proto
)

REM 尝试使用buf（如果安装）进行lint检查
where buf >nul 2>&1
if %ERRORLEVEL% equ 0 (
    echo ===== 第6步：使用buf进行lint检查 =====
    echo 运行buf lint检查...
    buf lint
    
    if %ERRORLEVEL% neq 0 (
        echo 警告: buf lint检查发现问题。
    ) else (
        echo buf lint检查通过。
    )
)

REM 尝试生成swagger.json (可能会失败，但不影响Go代码生成)
echo ===== 第7步：生成Swagger文档 =====
protoc --proto_path=third_party ^
       --proto_path=. ^
       --openapiv2_out=api/swagger ^
       --openapiv2_opt=logtostderr=true ^
       --openapiv2_opt=json_names_for_fields=true ^
       --openapiv2_opt=Mapi/example/v1/resource_example.proto=github.com/dormoron/phantasm/api/example/v1 ^
       api/example/v1/*.proto

if %ERRORLEVEL% neq 0 (
    echo 警告: Swagger文档生成失败，但Go代码应该已经生成成功。
)

echo 完成！
echo.
echo 生成结果:
echo - Google API: google.golang.org\genproto\googleapis\api
echo - Errors: github.com\dormoron\phantasm\third_party\errors
echo - Buf: github.com\dormoron\phantasm\third_party\buf
echo - 项目API: github.com\dormoron\phantasm\api\example\v1
echo - Swagger: %SWAGGER_OUT_DIR%
echo.

REM 显示生成的文件
echo Google API生成的文件:
dir google.golang.org\genproto\googleapis\api /s
echo.
echo Errors生成的文件:
dir github.com\dormoron\phantasm\third_party\errors\*.pb* 2>nul || echo 没有生成Errors文件
echo.
echo Buf生成的文件:
dir github.com\dormoron\phantasm\third_party\buf\*.pb* 2>nul || echo 没有生成Buf文件
echo.
echo 项目API生成的文件:
dir github.com\dormoron\phantasm\api\example\v1\*.pb* 2>nul || echo 没有生成API文件 