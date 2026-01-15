# T2I 模型测试脚本

本目录包含用于测试 Transformer-VNet 插件中三种文生图（T2I）模型的集成测试脚本。

## 环境配置

### 认证配置

测试脚本支持从 `.env` 文件读取认证信息。请按以下步骤配置：

1. **复制配置文件**：
   ```bash
   cp .env.example .env
   ```

2. **编辑 .env 文件**，根据需要配置认证信息：
   ```bash
   # Bearer Token 认证
   AUTH_TOKEN=your_bearer_token_here
   
   # API Key 认证
   API_KEY=your_api_key_here
   
   # 基础认证
   AUTH_USERNAME=your_username
   AUTH_PASSWORD=your_password
   
   # 自定义认证头
   CUSTOM_AUTH_HEADER=X-Custom-Auth
   CUSTOM_AUTH_VALUE=your_custom_value
   
   # API 配置
   API_URL=http://172.22.221.212/v1/images/generations
   TIMEOUT=60
   SERVICE_TIMEOUT=60
   ```

3. **支持的认证方式**：
   - **Bearer Token**: 设置 `AUTH_TOKEN`，会添加 `Authorization: Bearer <token>` 头
   - **API Key**: 设置 `API_KEY`，会添加 `X-API-Key: <key>` 头
   - **基础认证**: 设置 `AUTH_USERNAME` 和 `AUTH_PASSWORD`
   - **自定义头**: 设置 `CUSTOM_AUTH_HEADER` 和 `CUSTOM_AUTH_VALUE`

4. **多种认证方式**: 可以同时配置多种认证方式，脚本会自动应用所有配置的认证方法。

### 配置文件说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `AUTH_TOKEN` | Bearer Token 认证令牌 | 空 |
| `API_KEY` | API Key 认证密钥 | 空 |
| `AUTH_USERNAME` | 基础认证用户名 | 空 |
| `AUTH_PASSWORD` | 基础认证密码 | 空 |
| `CUSTOM_AUTH_HEADER` | 自定义认证头名称 | 空 |
| `CUSTOM_AUTH_VALUE` | 自定义认证头值 | 空 |
| `API_URL` | API 端点地址 | `http://localhost:10000/v1/images/generations` |
| `TIMEOUT` | 请求超时时间（秒） | 60 |
| `SERVICE_TIMEOUT` | 服务启动超时时间（秒） | 60 |

## 脚本说明

### test_all_models.py

Python 版本的集成测试脚本，功能完整，推荐使用。

**特性：**
- 自动读取 Makefile 配置
- 自动切换 Envoy 配置文件
- 支持三种模型：豆包 SeeDream 4.5、通义千问图像生成、Gemini 3 Image
- 详细的测试结果分析
- 优雅的错误处理和资源清理
- 支持信号处理（Ctrl+C 安全退出）

**依赖：**
- Python 3.6+
- requests 库：`pip install requests`
- .env 文件（可选，用于认证配置）

**使用方法：**
```bash
# 配置认证信息
cp .env.example .env
# 编辑 .env 文件填入认证信息

# 测试所有模型
python test_all_models.py

# 设置超时时间
python test_all_models.py --timeout 60
```

### test_all_models.sh

Shell 版本的集成测试脚本，无需 Python 环境。

**特性：**
- 纯 Shell 实现，无需额外依赖
- 自动切换 Envoy 配置
- 支持三种模型测试
- 彩色输出，易于阅读
- 自动资源清理

**依赖：**
- curl（必需）
- jq（可选，用于 JSON 格式化）
- bc（可选，用于精确时间计算）
- .env 文件（可选，用于认证配置）

**使用方法：**
```bash
# 给脚本添加执行权限
chmod +x test_all_models.sh

# 配置认证信息
cp .env.example .env
# 编辑 .env 文件填入认证信息

# 测试所有模型
./test_all_models.sh

# 设置超时时间
./test_all_models.sh --timeout 30 --service-timeout 60

# 查看帮助
./test_all_models.sh --help
```

## 测试流程

1. **停止当前服务**：确保没有其他服务占用端口
2. **切换配置**：使用 Makefile 中的目标切换到对应模型的 Envoy 配置
3. **启动服务**：启动对应模型的服务（后台模式）
4. **等待服务就绪**：检查服务健康状态
5. **执行测试用例**：发送预定义的测试请求
6. **分析结果**：解析响应，检查转换是否正确
7. **清理资源**：停止服务，准备下一个模型测试

## 测试用例

### 中文测试用例（豆包、通义千问）
1. **简单风景画**：生成一幅美丽的风景画，包含山川、河流和蓝天白云
2. **现代建筑**：设计一座现代化的摩天大楼，具有玻璃幕墙和独特的几何造型

### 英文测试用例（Gemini）
1. **Simple Landscape**：Generate a beautiful landscape painting with mountains, rivers, and blue sky with white clouds
2. **Modern Architecture**：Design a modern skyscraper with glass curtain walls and unique geometric shapes

## 模型配置

### 豆包 SeeDream 4.5
- **Model Header**: `volcengine/volcengine/doubao-seedream-4.5`
- **Model Type**: `t2i`
- **Make Target**: `dev-t2i-doubao-d`
- **Envoy Config**: `envoy.yaml`

### 通义千问图像生成
- **Model Header**: `aliyun/aliyun/qwen-image-plus`
- **Model Type**: `t2i`
- **Make Target**: `dev-t2i-qwen-d`
- **Envoy Config**: `envoy-qwen.yaml`

### Gemini 3 Image
- **Model Header**: `openrouter/auto/gemini-3-image`
- **Model Type**: `t2i`
- **Make Target**: `dev-t2i-gemini-d`
- **Envoy Config**: `envoy-gemini.yaml`

## 输出示例

```
🚀 开始测试模型: 豆包 SeeDream 4.5
Make 目标: dev-t2i-doubao-d
Envoy 配置: envoy.yaml

1. 停止当前服务...
2. 启动 豆包 SeeDream 4.5 服务...
ℹ️  等待服务启动 (超时: 60s)...
✅ 服务已启动

3. 执行测试用例...

测试用例 1/2: 简单风景画

============================================================
模型: 豆包 SeeDream 4.5
测试用例: 简单风景画
提示词: 生成一幅美丽的风景画，包含山川、河流和蓝天白云
============================================================
✅ 请求成功
状态码: 200
响应时间: 15.23s
响应模型: volcengine/volcengine/doubao-seedream-4.5
响应模型类型: t2i
请求ID: 1768313092
完成原因: stop
生成图片数量: 1
图片 1: https://example.com/images/doubao-seedream-4-5/example-image.jpeg
使用统计 - 图片数量: 1

📊 豆包 SeeDream 4.5 测试结果: 2/2 成功
```

## 错误处理

脚本包含完善的错误处理机制：

1. **服务启动失败**：自动重试或跳到下一个模型
2. **请求超时**：记录超时信息，继续测试
3. **响应解析错误**：显示原始响应内容
4. **信号中断**：优雅清理资源后退出
5. **依赖检查**：检查必需的命令和文件

## 注意事项

1. **认证配置**：确保 .env 文件中的认证信息正确配置
2. **端口占用**：确保 10000 端口未被其他服务占用
3. **Docker 环境**：确保 Docker 和 docker-compose 正常运行
4. **网络连接**：某些模型可能需要外网访问
5. **资源清理**：脚本会自动清理，但建议测试完成后手动检查
6. **日志查看**：如果测试失败，可以查看 docker-compose 日志：
   ```bash
   cd deploy_dev
   docker-compose logs
   ```

## 扩展测试

如需添加新的测试用例或模型，可以修改脚本中的相应配置：

1. **添加测试用例**：修改 `TEST_CASES` 数组
2. **添加新模型**：在 `MODELS` 配置中添加新模型信息
3. **自定义参数**：修改 `create_request_payload` 函数
4. **调整超时**：修改 `TIMEOUT` 和 `SERVICE_TIMEOUT` 变量

## 故障排除

### 常见问题

1. **认证失败**
   - 检查 .env 文件是否存在且配置正确
   - 验证认证信息是否有效
   - 确认认证方式是否匹配 API 要求

2. **服务启动超时**
   - 检查 Docker 服务状态
   - 查看端口是否被占用：`lsof -i :10000`
   - 检查 docker-compose 日志

3. **请求失败**
   - 确认 API 端点正确
   - 检查 Header 配置
   - 验证请求载荷格式
   - 检查认证信息是否正确

4. **响应解析错误**
   - 检查响应内容类型
   - 验证 JSON 格式
   - 查看错误状态码

### 调试技巧

1. **启用详细日志**：修改脚本中的日志级别
2. **单独测试**：可以手动启动服务后单独测试某个模型
3. **检查配置**：确认 Envoy 配置文件正确切换
4. **网络诊断**：使用 curl 直接测试 API 端点

## 贡献

如果发现问题或有改进建议，请：

1. 检查现有的 issue
2. 创建新的 issue 描述问题
3. 提交 Pull Request 修复问题

## 许可证

本测试脚本遵循项目的 Apache License 2.0 许可证。