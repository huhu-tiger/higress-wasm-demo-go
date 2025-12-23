#!/bin/bash

# AI 数据脱敏插件测试脚本
# 使用前确保已启动服务: docker-compose up

BASE_URL="http://localhost:10000"
echo "=========================================="
echo "AI 数据脱敏插件测试"
echo "=========================================="
echo ""

# 测试 1: OpenAI 格式 - 敏感词拦截（应该被拦截）
echo "【测试 1】OpenAI 格式 - 敏感词拦截测试"
echo "请求包含敏感词: '测试敏感词'"
echo "预期: 返回拦截消息"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "这是一个测试敏感词的请求"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 2: OpenAI 格式 - 正常请求（应该通过）
echo "【测试 2】OpenAI 格式 - 正常请求测试"
echo "请求不包含敏感词"
echo "预期: 正常转发到后端"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "这是一个正常的请求"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 3: JSONPath - 敏感词拦截
echo "【测试 3】JSONPath - 敏感词拦截测试"
echo "在指定 JSONPath 路径包含敏感词"
echo "预期: 返回拦截消息"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {
        "content": "这里包含违规内容"
      }
    ]
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 4: 手机号替换（不还原）
echo "【测试 4】手机号替换测试"
echo "请求包含手机号: 13812345678"
echo "预期: 手机号被替换为 ****"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "我的手机号是13812345678，请记住"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 5: IP 地址替换（可还原）
echo "【测试 5】IP 地址替换测试（可还原）"
echo "请求包含 IP: 192.168.1.100"
echo "预期: IP 被替换为 ***.***.***.***，响应中可还原"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "服务器地址是 192.168.1.100"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 6: 身份证号替换
echo "【测试 6】身份证号替换测试"
echo "请求包含身份证: 110101199001011234"
echo "预期: 身份证被替换为 ****"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "身份证号是 110101199001011234"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 7: API Key Hash 替换（可还原）
echo "【测试 7】API Key Hash 替换测试（可还原）"
echo "请求包含 API Key: sk-1234567890abcdef"
echo "预期: API Key 被 hash，响应中可还原"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "API Key 是 sk-1234567890abcdef"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "----------------------------------------"
echo ""

# 测试 8: 流式响应测试
echo "【测试 8】OpenAI 流式响应测试"
echo "流式请求，包含敏感词"
echo "预期: 流式响应被拦截"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "请回答关于测试敏感词的问题"
      }
    ],
    "stream": true
  }' | head -20
echo ""
echo "----------------------------------------"
echo ""

# 测试 9: 多个敏感词组合
echo "【测试 9】多个敏感词组合测试"
echo "请求包含多个敏感词和需要替换的内容"
curl -X POST "${BASE_URL}/post" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "我的手机是13812345678，IP是192.168.1.100，身份证是110101199001011234"
      }
    ],
    "stream": false
  }' | jq '.' 2>/dev/null || echo ""
echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="




