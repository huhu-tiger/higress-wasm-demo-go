#!/bin/bash

# Transformer-VNet T2I 模型集成测试脚本 (Shell 版本)
# 自动切换 Envoy 配置并测试三种 t2i 模型
# 支持从 .env 文件读取认证信息

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 加载 .env 文件
load_env() {
    local env_file="$PROJECT_ROOT/.env"
    if [ -f "$env_file" ]; then
        print_info "加载环境配置文件: $env_file"
        # 导出环境变量，忽略注释和空行
        set -a
        source <(grep -v '^#' "$env_file" | grep -v '^$')
        set +a
    else
        print_warning "未找到 .env 文件: $env_file"
        print_info "请复制 .env.example 为 .env 并配置认证信息"
        
        # 询问是否继续
        read -p "是否继续测试（无认证）？[y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "测试已取消"
            exit 0
        fi
    fi
}

# 默认配置
API_URL="${API_URL:-http://localhost:10000/v1/images/generations}"
TIMEOUT="${TIMEOUT:-30}"
SERVICE_TIMEOUT="${SERVICE_TIMEOUT:-60}"

# 认证配置
AUTH_TOKEN="${AUTH_TOKEN:-}"
API_KEY="${API_KEY:-}"
AUTH_USERNAME="${AUTH_USERNAME:-}"
AUTH_PASSWORD="${AUTH_PASSWORD:-}"
CUSTOM_AUTH_HEADER="${CUSTOM_AUTH_HEADER:-}"
CUSTOM_AUTH_VALUE="${CUSTOM_AUTH_VALUE:-}"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# 模型配置
declare -A MODELS
MODELS[doubao_model]="volcengine/volcengine/doubao-seedream-4.5"
MODELS[doubao_type]="t2i"
MODELS[doubao_name]="豆包 SeeDream 4.5"
MODELS[doubao_target]="dev-t2i-doubao-d"
MODELS[doubao_config]="envoy.yaml"

MODELS[qwen_model]="aliyun/aliyun/qwen-image-plus"
MODELS[qwen_type]="t2i"
MODELS[qwen_name]="通义千问图像生成"
MODELS[qwen_target]="dev-t2i-qwen-d"
MODELS[qwen_config]="envoy-qwen.yaml"

MODELS[gemini_model]="openrouter/auto/gemini-3-image"
MODELS[gemini_type]="t2i"
MODELS[gemini_name]="Gemini 3 Image"
MODELS[gemini_target]="dev-t2i-gemini-d"
MODELS[gemini_config]="envoy-gemini.yaml"

# 测试用例
TEST_CASES=(
    "生成一幅美丽的风景画，包含山川、河流和蓝天白云|简单风景画"
    "设计一座现代化的摩天大楼，具有玻璃幕墙和独特的几何造型|现代建筑"
)

GEMINI_TEST_CASES=(
    "Generate a beautiful landscape painting with mountains, rivers, and blue sky with white clouds|Simple Landscape"
    "Design a modern skyscraper with glass curtain walls and unique geometric shapes|Modern Architecture"
)

# 打印配置信息
print_config() {
    print_header "配置信息"
    echo "  API URL: $API_URL"
    echo "  请求超时: ${TIMEOUT}s"
    echo "  服务启动超时: ${SERVICE_TIMEOUT}s"
    
    # 打印认证配置（隐藏敏感信息）
    local auth_methods=()
    if [ -n "$AUTH_TOKEN" ]; then
        auth_methods+=("Bearer Token: ${AUTH_TOKEN:0:10}...")
    fi
    if [ -n "$API_KEY" ]; then
        auth_methods+=("API Key: ${API_KEY:0:10}...")
    fi
    if [ -n "$AUTH_USERNAME" ] && [ -n "$AUTH_PASSWORD" ]; then
        auth_methods+=("Basic Auth: $AUTH_USERNAME:***")
    fi
    if [ -n "$CUSTOM_AUTH_HEADER" ] && [ -n "$CUSTOM_AUTH_VALUE" ]; then
        auth_methods+=("Custom Header: $CUSTOM_AUTH_HEADER")
    fi
    
    if [ ${#auth_methods[@]} -gt 0 ]; then
        printf "  认证方式: "
        IFS=', '; echo "${auth_methods[*]}"
    else
        echo "  认证方式: 无"
    fi
    echo ""
}

# 构建认证头参数
build_auth_headers() {
    local headers=()
    
    # Bearer Token 认证
    if [ -n "$AUTH_TOKEN" ]; then
        headers+=("-H" "Authorization: Bearer $AUTH_TOKEN")
    fi
    
    # API Key 认证
    if [ -n "$API_KEY" ]; then
        headers+=("-H" "X-API-Key: $API_KEY")
    fi
    
    # 自定义认证头
    if [ -n "$CUSTOM_AUTH_HEADER" ] && [ -n "$CUSTOM_AUTH_VALUE" ]; then
        headers+=("-H" "$CUSTOM_AUTH_HEADER: $CUSTOM_AUTH_VALUE")
    fi
    
    # 返回头参数数组
    printf '%s\n' "${headers[@]}"
}

# 构建认证参数（用于 curl）
build_auth_params() {
    local params=()
    
    # 基础认证
    if [ -n "$AUTH_USERNAME" ] && [ -n "$AUTH_PASSWORD" ]; then
        params+=("--user" "$AUTH_USERNAME:$AUTH_PASSWORD")
    fi
    
    # 返回参数数组
    printf '%s\n' "${params[@]}"
}
# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_header() {
    echo -e "${PURPLE}🚀 $1${NC}"
}

# 清理函数
cleanup() {
    print_info "正在清理资源..."
    cd "$PROJECT_ROOT"
    make stop >/dev/null 2>&1 || true
}

# 信号处理
trap cleanup EXIT INT TERM

# 等待服务启动
wait_for_service() {
    local timeout=$1
    print_info "等待服务启动 (超时: ${timeout}s)..."
    
    local start_time=$(date +%s)
    local current_time
    
    # 构建认证参数
    local auth_headers=()
    readarray -t auth_headers < <(build_auth_headers)
    local auth_params=()
    readarray -t auth_params < <(build_auth_params)
    
    while true; do
        current_time=$(date +%s)
        if [ $((current_time - start_time)) -ge $timeout ]; then
            print_error "服务启动超时 (${timeout}s)"
            
            # 超时后检查 docker-compose 状态
            print_info "检查 Docker Compose 状态..."
            cd "$PROJECT_ROOT/deploy_dev"
            if command -v docker-compose >/dev/null 2>&1; then
                echo "Docker Compose 状态:"
                docker-compose ps || true
                echo ""
                echo "最近的日志:"
                docker-compose logs --tail=20 || true
            fi
            cd "$PROJECT_ROOT"
            
            return 1
        fi
        
        # 构建 curl 命令 - 尝试连接主端点
        local curl_cmd=("curl" "-s" "--connect-timeout" "3" "--max-time" "5" "http://localhost:10000")
        curl_cmd+=("${auth_headers[@]}")
        curl_cmd+=("${auth_params[@]}")
        
        # 执行 curl 并检查退出码
        # 只要不是连接拒绝（7）就认为服务启动了
        local curl_output
        curl_output=$("${curl_cmd[@]}" 2>&1)
        local curl_exit_code=$?
        
        # curl 退出码说明:
        # 0: 成功
        # 7: 连接失败 (Connection refused)
        # 28: 超时
        # 其他: 各种 HTTP 错误，但服务在运行
        if [ $curl_exit_code -eq 0 ] || [ $curl_exit_code -ne 7 ]; then
            # 如果不是连接拒绝，说明服务已经启动
            if [ $curl_exit_code -eq 0 ]; then
                print_success "服务已启动 (HTTP 响应正常)"
            else
                print_success "服务已启动 (检测到服务响应)"
            fi
            return 0
        fi
        
        echo -n "."
        sleep 2
    done
}

# 创建请求载荷
create_doubao_payload() {
    local prompt="$1"
    cat <<EOF
{
    "model": "${MODELS[doubao_model]}",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "$prompt"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "size": "1024*1024",
        "watermark": false,
        "n": 1
    },
    "extra_body": {
        "doubao_seedream": {
            "sequential_image_generation": "auto",
            "response_format": "url",
            "stream": false
        }
    }
}
EOF
}

create_qwen_payload() {
    local prompt="$1"
    cat <<EOF
{
    "model": "${MODELS[qwen_model]}",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "$prompt"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "size": "1024*1024",
        "watermark": false,
        "prompt_extend": true,
        "n": 1
    }
}
EOF
}

create_gemini_payload() {
    local prompt="$1"
    cat <<EOF
{
    "model": "${MODELS[gemini_model]}",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "$prompt"
                    }
                ]
            }
        ]
    },
    "parameters": {},
    "extra_body": {
        "gemini_3_image": {
            "modalities": ["image"],
            "provider": {
                "only": ["Google"]
            }
        }
    }
}
EOF
}

# 发送请求并分析结果
send_request() {
    local model_key="$1"
    local payload="$2"
    local test_name="$3"
    
    local model_name="${MODELS[${model_key}_name]}"
    local model_id="${MODELS[${model_key}_model]}"
    local model_type="${MODELS[${model_key}_type]}"
    
    print_info "测试 $model_name - $test_name"
    
    # 构建认证参数
    local auth_headers=()
    readarray -t auth_headers < <(build_auth_headers)
    local auth_params=()
    readarray -t auth_params < <(build_auth_params)
    
    # 打印认证信息
    if [ ${#auth_headers[@]} -gt 0 ]; then
        print_info "使用认证头: $(printf '%s ' "${auth_headers[@]}" | grep -o '\-H [^:]*:' | sed 's/-H //g' | sed 's/://g' | tr '\n' ' ')"
    fi
    if [ ${#auth_params[@]} -gt 0 ]; then
        print_info "使用基础认证: $AUTH_USERNAME:***"
    fi
    
    local start_time=$(date +%s.%N)
    
    # 构建完整的 curl 命令
    local curl_cmd=(
        "curl" "-s" "-w" "\n%{http_code}\n%{time_total}"
        "-X" "POST" "$API_URL"
        "-H" "Content-Type: application/json"
        "-H" "model: $model_id"
        "-H" "model_type: $model_type"
        "-d" "$payload"
        "--max-time" "$TIMEOUT"
    )
    
    # 添加认证头
    curl_cmd+=("${auth_headers[@]}")
    # 添加认证参数
    curl_cmd+=("${auth_params[@]}")
    
    local response=$("${curl_cmd[@]}")
    
    local end_time=$(date +%s.%N)
    local response_time
    if command -v bc >/dev/null 2>&1; then
        response_time=$(echo "$end_time - $start_time" | bc)
    else
        response_time="N/A"
    fi
    
    # 解析响应
    local body=$(echo "$response" | head -n -2)
    local http_code=$(echo "$response" | tail -n 2 | head -n 1)
    local curl_time=$(echo "$response" | tail -n 1)
    
    echo "========================================"
    echo "模型: $model_name"
    echo "测试用例: $test_name"
    echo "========================================"
    echo "状态码: $http_code"
    echo "响应时间: ${response_time}s"
    
    # 分析响应内容
    if command -v jq >/dev/null 2>&1; then
        echo "响应内容:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        
        # 提取关键信息
        local request_id=$(echo "$body" | jq -r '.request_id // empty' 2>/dev/null)
        local finish_reason=$(echo "$body" | jq -r '.output.choices[0].finish_reason // empty' 2>/dev/null)
        local image_count=$(echo "$body" | jq -r '.usage.image_count // empty' 2>/dev/null)
        local error_code=$(echo "$body" | jq -r '.code // empty' 2>/dev/null)
        
        if [ -n "$request_id" ]; then
            echo "请求ID: $request_id"
        fi
        
        if [ -n "$finish_reason" ]; then
            echo "完成原因: $finish_reason"
        fi
        
        if [ -n "$image_count" ]; then
            echo "生成图片数量: $image_count"
        fi
        
        if [ -n "$error_code" ]; then
            echo "错误码: $error_code"
        fi
    else
        echo "响应内容: $body"
    fi
    
    echo ""
    
    if [ "$http_code" = "200" ]; then
        print_success "$model_name 测试成功"
        return 0
    else
        print_error "$model_name 测试失败 (HTTP $http_code)"
        return 1
    fi
}

# 测试单个模型
test_model() {
    local model_key="$1"
    local model_name="${MODELS[${model_key}_name]}"
    local make_target="${MODELS[${model_key}_target]}"
    local envoy_config="${MODELS[${model_key}_config]}"
    
    print_header "开始测试模型: $model_name"
    echo "Make 目标: $make_target"
    echo "Envoy 配置: $envoy_config"
    echo ""
    
    # 1. 停止当前服务
    print_info "1. 停止当前服务..."
    cd "$PROJECT_ROOT"
    make stop >/dev/null 2>&1 || true
    sleep 3
    
    # 2. 启动新的服务配置
    print_info "2. 启动 $model_name 服务..."
    if ! make "$make_target" >/dev/null 2>&1; then
        print_error "启动 $model_name 服务失败"
        return 1
    fi
    
    # 3. 等待服务启动
    if ! wait_for_service $SERVICE_TIMEOUT; then
        print_error "$model_name 服务启动超时"
        return 1
    fi
    
    # 4. 执行测试用例
    print_info "3. 执行测试用例..."
    local success_count=0
    local total_count
    
    if [ "$model_key" = "gemini" ]; then
        total_count=${#GEMINI_TEST_CASES[@]}
        for test_case in "${GEMINI_TEST_CASES[@]}"; do
            IFS='|' read -r prompt name <<< "$test_case"
            local payload=$(create_gemini_payload "$prompt")
            if send_request "$model_key" "$payload" "$name"; then
                ((success_count++))
            fi
            sleep 3
        done
    else
        total_count=${#TEST_CASES[@]}
        for test_case in "${TEST_CASES[@]}"; do
            IFS='|' read -r prompt name <<< "$test_case"
            local payload
            if [ "$model_key" = "doubao" ]; then
                payload=$(create_doubao_payload "$prompt")
            else
                payload=$(create_qwen_payload "$prompt")
            fi
            if send_request "$model_key" "$payload" "$name"; then
                ((success_count++))
            fi
            sleep 3
        done
    fi
    
    echo "========================================"
    print_info "$model_name 测试结果: $success_count/$total_count 成功"
    echo "========================================"
    
    if [ $success_count -eq $total_count ]; then
        return 0
    else
        return 1
    fi
}

# 主函数
main() {
    # 加载环境变量
    load_env
    
    # 打印配置信息
    print_config
    
    print_header "Transformer-VNet T2I 模型集成测试脚本"
    echo "项目根目录: $PROJECT_ROOT"
    echo "API 端点: $API_URL"
    echo "请求超时: ${TIMEOUT}s"
    echo "服务启动超时: ${SERVICE_TIMEOUT}s"
    echo ""
    
    # 检查依赖
    if ! command -v curl >/dev/null 2>&1; then
        print_error "curl 命令未找到，请先安装 curl"
        exit 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        print_warning "jq 命令未找到，JSON 格式化将不可用"
    fi
    
    if ! command -v bc >/dev/null 2>&1; then
        print_warning "bc 命令未找到，响应时间计算可能不准确"
    fi
    
    # 检查项目结构
    if [ ! -f "$PROJECT_ROOT/Makefile" ]; then
        print_error "Makefile 不存在: $PROJECT_ROOT/Makefile"
        exit 1
    fi
    
    if [ ! -d "$PROJECT_ROOT/deploy_dev" ]; then
        print_error "部署目录不存在: $PROJECT_ROOT/deploy_dev"
        exit 1
    fi
    
    # 测试所有模型
    local models=("doubao" "qwen" "gemini")
    local results=()
    local success_count=0
    
    for model_key in "${models[@]}"; do
        echo ""
        echo "================================================================================"
        if test_model "$model_key"; then
            results+=("${MODELS[${model_key}_name]}: ✅ 成功")
            ((success_count++))
        else
            results+=("${MODELS[${model_key}_name]}: ❌ 失败")
            print_warning "${MODELS[${model_key}_name]} 测试失败，继续测试下一个模型..."
        fi
        echo "================================================================================"
        sleep 5
    done
    
    # 打印测试总结
    echo ""
    echo "================================================================================"
    print_header "测试总结"
    echo "================================================================================"
    
    for result in "${results[@]}"; do
        echo "$result"
    done
    
    echo ""
    echo "总体结果: $success_count/${#models[@]} 模型测试成功"
    
    if [ $success_count -eq ${#models[@]} ]; then
        print_success "🎉 所有模型测试通过!"
        exit 0
    else
        print_error "⚠️  部分模型测试失败，请检查日志"
        exit 1
    fi
}

# 显示帮助信息
show_help() {
    echo "Transformer-VNet T2I 模型集成测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -t, --timeout  设置请求超时时间（秒），默认30秒"
    echo "  -s, --service-timeout  设置服务启动超时时间（秒），默认60秒"
    echo ""
    echo "说明:"
    echo "  此脚本会自动切换 Envoy 配置并测试三种 t2i 模型："
    echo "  - 豆包 SeeDream 4.5"
    echo "  - 通义千问图像生成"
    echo "  - Gemini 3 Image"
    echo ""
    echo "  脚本会读取 Makefile 中的配置，自动启动和停止服务。"
    echo "  认证信息从 .env 文件中读取。"
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -s|--service-timeout)
            SERVICE_TIMEOUT="$2"
            shift 2
            ;;
        *)
            print_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 运行主函数
main