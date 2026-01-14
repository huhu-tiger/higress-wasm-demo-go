#!/bin/bash

# 快速切换 Envoy 配置脚本

CONFIG_TYPE=$1

if [ -z "$CONFIG_TYPE" ]; then
    echo "用法: $0 [qwen|doubao|gemini]"
    echo ""
    echo "选项:"
    echo "  qwen   - 使用 qwen-image-plus 配置（无需转换）"
    echo "  doubao - 使用 doubao-seedream-4.5 配置（默认）"
    echo "  gemini - 使用 gemini-3-image 配置"
    exit 1
fi

case $CONFIG_TYPE in
    qwen)
        CONFIG_FILE="t2i/envoy-qwen.yaml"
        echo "切换到 qwen-image-plus 配置..."
        ;;
    doubao)
        CONFIG_FILE="t2i/envoy.yaml"
        echo "切换到 doubao-seedream-4.5 配置..."
        ;;
    gemini)
        CONFIG_FILE="t2i/envoy-gemini.yaml"
        echo "切换到 gemini-3-image 配置..."
        ;;
    *)
        echo "错误: 未知的配置类型 '$CONFIG_TYPE'"
        echo "用法: $0 [qwen|doubao|gemini]"
        exit 1
        ;;
esac

# 更新 docker-compose.yml
# 只替换路径部分，保留 YAML 格式（缩进和列表标记）
# 处理标准格式：- ./t2i/envoy-*.yaml:/etc/envoy/envoy.yaml
sed -i.bak "s|- .*envoy.*\.yaml:/etc/envoy/envoy.yaml|- ./${CONFIG_FILE}:/etc/envoy/envoy.yaml|" docker-compose.yml
# 处理错误格式：./t2i/envoy-*.yaml:/etc/envoy/envoy.yaml（缺少列表标记）
sed -i.bak "s|^[[:space:]]*\./.*envoy.*\.yaml:/etc/envoy/envoy.yaml|    - ./${CONFIG_FILE}:/etc/envoy/envoy.yaml|" docker-compose.yml

echo "配置已更新为: ${CONFIG_FILE}"
echo ""
echo "请运行以下命令重启服务:"
echo "  docker-compose down"
echo "  docker-compose up -d"
