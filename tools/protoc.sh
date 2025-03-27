#!/bin/bash

set -e

# 当前脚本路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 项目根目录
ROOT_DIR="$(dirname "${SCRIPT_DIR}")"
# API目录
API_DIR="${ROOT_DIR}/api"
# API文档输出目录
SWAGGER_OUT_DIR="${ROOT_DIR}/api/swagger"

# 确保输出目录存在
mkdir -p "${SWAGGER_OUT_DIR}"

# 查找所有的proto文件
PROTO_FILES=$(find "${API_DIR}" -name "*.proto")

# 检查是否安装了所需的插件
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is not installed. Please install it first."
        echo "You can use: go install $2"
        exit 1
    fi
}

check_command "protoc" "protobuf compiler"
check_command "protoc-gen-go" "google.golang.org/protobuf/cmd/protoc-gen-go@latest"
check_command "protoc-gen-go-grpc" "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
check_command "protoc-gen-validate" "github.com/envoyproxy/protoc-gen-validate@latest"
check_command "protoc-gen-openapiv2" "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest"

for PROTO_FILE in ${PROTO_FILES}; do
    echo "Processing: ${PROTO_FILE}"
    
    # 获取proto文件的目录
    PROTO_DIR=$(dirname "${PROTO_FILE}")
    # 相对于API_DIR的路径
    REL_PATH=${PROTO_DIR#${API_DIR}/}
    
    # 生成Go代码
    protoc --proto_path="${API_DIR}" \
           --proto_path="${ROOT_DIR}" \
           --proto_path="${ROOT_DIR}/third_party" \
           --go_out="${ROOT_DIR}" \
           --go_opt=paths=source_relative \
           --go-grpc_out="${ROOT_DIR}" \
           --go-grpc_opt=paths=source_relative \
           --validate_out="lang=go,paths=source_relative:${ROOT_DIR}" \
           "${PROTO_FILE}"
    
    # 生成swagger.json
    protoc --proto_path="${API_DIR}" \
           --proto_path="${ROOT_DIR}" \
           --proto_path="${ROOT_DIR}/third_party" \
           --openapiv2_out="${SWAGGER_OUT_DIR}" \
           --openapiv2_opt=logtostderr=true \
           --openapiv2_opt=json_names_for_fields=true \
           "${PROTO_FILE}"
done

echo "All proto files processed successfully!"
echo "Go code has been generated in the respective directories."
echo "Swagger documentation has been generated in ${SWAGGER_OUT_DIR}" 