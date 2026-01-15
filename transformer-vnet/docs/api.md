# API 使用说明文档

本文档详细说明了 Transformer-VNet 插件支持的各个 AI 模型的统一 API 格式、原始格式以及参数对应关系。

## 更新日志

| 日期 | 版本 | 更新内容 |
|------|------|----------|
| 2026-01-15 | v1.0.0 | 初始版本，支持豆包 SeeDream 4.5、通义千问图像生成、Gemini 3 Image |

---

## 快速开始

### API 端点

```
POST http://172.22.221.212/v1/images/generations
```

### 必需的 HTTP Headers

所有请求必须包含以下 Header：

| Header | 说明 | 示例值 |
|--------|------|--------|
| `Content-Type` | 请求内容类型 | `application/json` |
| `model` | 模型标识符 | `volcengine/volcengine/doubao-seedream-4.5` |
| `model_type` | 模型类型 | `t2i` |

### 快速示例

```bash
curl -X POST http://172.22.221.212/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "model: volcengine/volcengine/doubao-seedream-4.5" \
  -H "model_type: t2i" \
  -d '{
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "input": {
      "messages": [
        {
          "role": "user",
          "content": [{"text": "生成一幅美丽的风景画"}]
        }
      ]
    },
    "parameters": {
      "size": "1024*1024",
      "n": 1
    }
  }'
```

---

## 目录

- [快速开始](#快速开始)
- [文生图 (Text-to-Image)](#文生图-text-to-image)
  - [豆包 SeeDream 4.5](#豆包-seedream-45)
  - [通义千问图像生成](#通义千问图像生成)
  - [Gemini 3 Image](#gemini-3-image)
  - [使用示例](#使用示例)
  - [注意事项](#注意事项)
- [统一错误格式](#统一错误格式)
- [更新日志](#更新日志)

---

# 文生图 (Text-to-Image)

## 通用说明

所有文生图模型使用统一的请求和响应格式。

**API 端点**: `http://172.22.221.212/v1/images/generations`

**必需的 HTTP Headers**:
- `Content-Type: application/json`
- `model`: 模型标识符（如 `volcengine/volcengine/doubao-seedream-4.5`）
- `model_type`: 模型类型（固定为 `t2i`）

**重要提示**: 请求时必须在 HTTP Header 中指定 `model` 和 `model_type`，否则插件无法识别模型类型进行转换。

---

## 豆包 SeeDream 4.5

### 模型标识

- **Model Header**: `volcengine/volcengine/doubao-seedream-4.5`
- **Model Type**: `t2i`
- **官方文档**: [豆包 SeeDream API](https://www.volcengine.com/docs/82379/1541523?lang=zh)

### 统一请求格式

```json
{
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "生成一组共1张连贯插画，核心为同一庭院一角的四季变迁"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "size": "1328*1328",
        "watermark": true,
        "n": 1,
        "seed": 12345
    },
    "extra_body": {
        "doubao_seedream": {
            "sequential_image_generation": "auto",
            "response_format": "url",
            "stream": false
        }
    }
}
```

### 统一请求参数说明

| 参数路径 | 类型 | 必填 | 说明 | 示例值 |
|---------|------|------|------|--------|
| `model` | string | 是 | 模型标识符 | `"volcengine/volcengine/doubao-seedream-4.5"` |
| `input.messages[0].role` | string | 是 | 角色类型，固定为 `"user"` | `"user"` |
| `input.messages[0].content[0].text` | string | 是 | 文本提示词，描述期望生成的图像内容 | `"生成一幅美丽的风景画"` |
| `parameters.size` | string | 否 | 图像尺寸，格式为 `"宽*高"`（统一使用 Qwen 格式） | `"1328*1328"` |
| `parameters.watermark` | boolean | 否 | 是否添加水印 | `true` / `false` |
| `parameters.n` | integer | 否 | 生成图片数量 | `1` 到 `15` |
| `parameters.seed` | integer | 否 | 随机数种子 | `0` 到 `2147483647` |
| `extra_body.doubao_seedream.sequential_image_generation` | string | 否 | 是否启用序列图片生成 | `"disabled"` / `"auto"` |
| `extra_body.doubao_seedream.response_format` | string | 否 | 响应格式 | `"url"` / `"b64_json"` |
| `extra_body.doubao_seedream.stream` | boolean | 否 | 是否启用流式返回 | `true` / `false` |

### 原始请求格式（豆包 API）

```json
{
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "prompt": "生成一组共1张连贯插画，核心为同一庭院一角的四季变迁",
    "size": "1328x1328",
    "watermark": true,
    "seed": 12345,
    "sequential_image_generation": "auto",
    "sequential_image_generation_options": {
        "max_images": 1
    },
    "response_format": "url",
    "stream": false
}
```

### 请求参数对应关系

| 统一格式 | 原始格式 | 转换说明 |
|---------|---------|---------|
| `input.messages[0].content[0].text` | `prompt` | 直接提取文本内容 |
| `parameters.size` (`"1328*1328"`) | `size` (`"1328x1328"`) | 格式转换：`*` → `x`<br>特殊映射：正方形约 4M 像素 → `"2K"`<br>正方形约 16M 像素 → `"4K"` |
| `parameters.watermark` | `watermark` | 直接映射 |
| `parameters.seed` | `seed` | 直接映射 |
| `parameters.n` | `sequential_image_generation_options.max_images` | 映射到嵌套对象 |
| `extra_body.doubao_seedream.sequential_image_generation` | `sequential_image_generation` | 提升到顶层 |
| `extra_body.doubao_seedream.response_format` | `response_format` | 提升到顶层 |
| `extra_body.doubao_seedream.stream` | `stream` | 提升到顶层 |

### 统一响应格式

```json
{
    "request_id": "1768313092",
    "output": {
        "choices": [
            {
                "finish_reason": "stop",
                "message": {
                    "role": "assistant",
                    "content": [
                        {
                            "image": "https://example.com/images/doubao-seedream-4-5/example-image.jpeg"
                        }
                    ]
                }
            }
        ],
        "task_metric": {
            "FAILED": 0,
            "SUCCEEDED": 1,
            "TOTAL": 1
        }
    },
    "usage": {
        "image_count": 1
    }
}
```

### 统一响应参数说明

| 参数路径 | 类型 | 说明 |
|---------|------|------|
| `request_id` | string | 请求唯一标识符 |
| `output.choices[0].finish_reason` | string | 完成原因，通常为 `"stop"` |
| `output.choices[0].message.role` | string | 角色类型，固定为 `"assistant"` |
| `output.choices[0].message.content[].image` | string | 生成图像的 URL 或 Base64 编码 |
| `output.task_metric.FAILED` | integer | 失败任务数 |
| `output.task_metric.SUCCEEDED` | integer | 成功任务数 |
| `output.task_metric.TOTAL` | integer | 总任务数 |
| `usage.image_count` | integer | 生成图片数量 |

### 原始响应格式（豆包 API）

```json
{
    "model": "doubao-seedream-4-5-251128",
    "created": 1768313092,
    "data": [
        {
            "url": "https://example.com/images/doubao-seedream-4-5/example-image.jpeg",
            "size": "2048x2048"
        }
    ],
    "usage": {
        "generated_images": 1,
        "output_tokens": 16384,
        "total_tokens": 16384
    }
}
```

### 响应参数对应关系

| 统一格式 | 原始格式 | 转换说明 |
|---------|---------|---------|
| `request_id` | `created` | 时间戳转字符串 |
| `output.choices[0].finish_reason` | - | 固定值 `"stop"` |
| `output.choices[0].message.role` | - | 固定值 `"assistant"` |
| `output.choices[0].message.content[0].image` | `data[0].url` 或 `data[0].b64_json` | 提取图片 URL 或 Base64 |
| `output.task_metric.SUCCEEDED` | `usage.generated_images` | 映射生成数量 |
| `output.task_metric.TOTAL` | `usage.generated_images` | 映射生成数量 |
| `output.task_metric.FAILED` | - | 固定值 `0` |
| `usage.image_count` | `usage.generated_images` | 直接映射 |

---

## 通义千问图像生成

### 模型标识

- **Model Header**: `aliyun/aliyun/qwen-image-plus`
- **Model Type**: `t2i`
- **官方文档**: [通义千问图像生成 API](https://bailian.console.aliyun.com/)

### 统一请求格式

```json
{
    "model": "aliyun/aliyun/qwen-image-plus",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "一副典雅庄重的对联悬挂于厅堂之中"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "size": "1328*1328",
        "negative_prompt": "",
        "prompt_extend": true,
        "watermark": false,
        "n": 1,
        "seed": 12345
    }
}
```

### 统一请求参数说明

| 参数路径 | 类型 | 必填 | 说明 | 示例值 |
|---------|------|------|------|--------|
| `model` | string | 是 | 模型标识符 | `"aliyun/aliyun/qwen-image-plus"` |
| `input.messages[0].role` | string | 是 | 角色类型，固定为 `"user"` | `"user"` |
| `input.messages[0].content[0].text` | string | 是 | 文本提示词 | `"一副典雅庄重的对联"` |
| `parameters.size` | string | 否 | 图像尺寸，格式为 `"宽*高"` | `"1328*1328"` / `"1664*928"` / `"928*1664"` / `"1472*1104"` / `"1104*1472"` |
| `parameters.negative_prompt` | string | 否 | 反向提示词 | `"低质量, 模糊"` |
| `parameters.prompt_extend` | boolean | 否 | 是否启用提示词智能扩展 | `true` / `false` |
| `parameters.watermark` | boolean | 否 | 是否添加水印 | `true` / `false` |
| `parameters.n` | integer | 否 | 生成图片数量 | `1` 到 `6` |
| `parameters.seed` | integer | 否 | 随机数种子 | `0` 到 `2147483647` |
| `parameters.steps` | integer | 否 | 推理步数 | `1` 到 `50`，推荐 `20` |
| `parameters.guidance_scale` | float | 否 | 引导强度 | `1.0` 到 `10.0`，推荐 `3.5` |

### 原始请求格式（通义千问 API）

通义千问使用的就是统一格式，无需转换。

```json
{
    "model": "aliyun/aliyun/qwen-image-plus",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "一副典雅庄重的对联悬挂于厅堂之中"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "size": "1328*1328",
        "negative_prompt": "",
        "prompt_extend": true,
        "watermark": false,
        "n": 1
    }
}
```

### 请求参数对应关系

通义千问格式即为统一格式，无需转换，所有参数保持一致。

### 统一响应格式

```json
{
    "request_id": "8cbed912-b740-4786-b770-ddf664508c24",
    "output": {
        "choices": [
            {
                "finish_reason": "stop",
                "message": {
                    "role": "assistant",
                    "content": [
                        {
                            "image": "https://dashscope-result-wlcb-acdr-1.oss-cn-wulanchabu-acdr-1.aliyuncs.com/..."
                        }
                    ]
                }
            }
        ],
        "task_metric": {
            "FAILED": 0,
            "SUCCEEDED": 1,
            "TOTAL": 1
        }
    },
    "usage": {
        "image_count": 1,
        "height": 1328,
        "width": 1328
    }
}
```

### 统一响应参数说明

| 参数路径 | 类型 | 说明 |
|---------|------|------|
| `request_id` | string | 请求唯一标识符 |
| `output.choices[0].finish_reason` | string | 完成原因 |
| `output.choices[0].message.role` | string | 角色类型，固定为 `"assistant"` |
| `output.choices[0].message.content[].image` | string | 生成图像的 URL |
| `output.task_metric.FAILED` | integer | 失败任务数 |
| `output.task_metric.SUCCEEDED` | integer | 成功任务数 |
| `output.task_metric.TOTAL` | integer | 总任务数 |
| `usage.image_count` | integer | 生成图片数量 |
| `usage.height` | integer | 图像高度（像素） |
| `usage.width` | integer | 图像宽度（像素） |

### 原始响应格式（通义千问 API）

通义千问使用的就是统一格式，无需转换。

```json
{
    "output": {
        "choices": [
            {
                "finish_reason": "stop",
                "message": {
                    "content": [
                        {
                            "image": "https://dashscope-result-wlcb-acdr-1.oss-cn-wulanchabu-acdr-1.aliyuncs.com/..."
                        }
                    ],
                    "role": "assistant"
                }
            }
        ],
        "task_metric": {
            "FAILED": 0,
            "SUCCEEDED": 1,
            "TOTAL": 1
        }
    },
    "usage": {
        "height": 1328,
        "image_count": 1,
        "width": 1328
    },
    "request_id": "8cbed912-b740-4786-b770-ddf664508c24"
}
```

### 响应参数对应关系

通义千问格式即为统一格式，无需转换，所有参数保持一致。

---

## Gemini 3 Image

### 模型标识

- **Model Header**: `openrouter/auto/gemini-3-image`
- **Model Type**: `t2i`
- **官方文档**: [Gemini 3 Image API](https://openrouter.ai/google/gemini-3-pro-image-preview/api)

### 统一请求格式

```json
{
    "model": "openrouter/auto/gemini-3-image",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "Generate a beautiful sunset over mountains"
                    }
                ]
            }
        ]
    },
    "parameters": {},
    "extra_body": {
        "gemini_3_image": {
            "modalities": ["image", "text"],
            "provider": {
                "only": ["Google"]
            }
        }
    }
}
```

### 统一请求参数说明

| 参数路径 | 类型 | 必填 | 说明 | 示例值 |
|---------|------|------|------|--------|
| `model` | string | 是 | 模型标识符 | `"openrouter/auto/gemini-3-image"` |
| `input.messages[0].role` | string | 是 | 角色类型 | `"user"` / `"assistant"` / `"system"` |
| `input.messages[0].content[0].text` | string | 是 | 文本提示词 | `"Generate a beautiful sunset"` |
| `extra_body.gemini_3_image.modalities` | array | 是 | 输出模态类型 | `["image", "text"]` / `["image"]` |
| `extra_body.gemini_3_image.provider.only` | array | 否 | 限制使用的提供商 | `["Google"]` |

### 原始请求格式（Gemini API）

```json
{
    "model": "openrouter/auto/gemini-3-image",
    "messages": [
        {
            "role": "user",
            "content": "Generate a beautiful sunset over mountains"
        }
    ],
    "modalities": ["image", "text"],
    "provider": {
        "only": ["Google"]
    }
}
```

### 请求参数对应关系

| 统一格式 | 原始格式 | 转换说明 |
|---------|---------|---------|
| `input.messages[0].role` | `messages[0].role` | 直接提取 |
| `input.messages[0].content[0].text` | `messages[0].content` | 提取文本内容，转为字符串 |
| `extra_body.gemini_3_image.modalities` | `modalities` | 提升到顶层 |
| `extra_body.gemini_3_image.provider` | `provider` | 提升到顶层 |

### 统一响应格式

```json
{
    "request_id": "gen-1768313027-nwX1nWNYMC56FXn2jLuT",
    "output": {
        "choices": [
            {
                "finish_reason": "stop",
                "message": {
                    "role": "assistant",
                    "content": [
                        {
                            "text": "I've generated a beautiful sunset image for you."
                        },
                        {
                            "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAABYAAAAMACAIAAAASU1SbAAAA..."
                        }
                    ]
                }
            }
        ],
        "task_metric": {
            "FAILED": 0,
            "SUCCEEDED": 1,
            "TOTAL": 1
        }
    },
    "usage": {
        "image_count": 1
    }
}
```

### 统一响应参数说明

| 参数路径 | 类型 | 说明 |
|---------|------|------|
| `request_id` | string | 请求唯一标识符 |
| `output.choices[0].finish_reason` | string | 完成原因 |
| `output.choices[0].message.role` | string | 角色类型，固定为 `"assistant"` |
| `output.choices[0].message.content[].text` | string | 文本内容（可选） |
| `output.choices[0].message.content[].image` | string | 生成图像的 Base64 编码 |
| `output.task_metric.FAILED` | integer | 失败任务数 |
| `output.task_metric.SUCCEEDED` | integer | 成功任务数 |
| `output.task_metric.TOTAL` | integer | 总任务数 |
| `usage.image_count` | integer | 生成图片数量 |

### 原始响应格式（Gemini API）

```json
{
    "id": "gen-1768313027-nwX1nWNYMC56FXn2jLuT",
    "provider": "Google",
    "model": "google/gemini-3-pro-image-preview",
    "object": "chat.completion",
    "created": 1768313028,
    "choices": [
        {
            "index": 0,
            "finish_reason": "stop",
            "message": {
                "role": "assistant",
                "content": "",
                "images": [
                    {
                        "type": "image_url",
                        "image_url": {
                            "url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAABYAAAAMACAIAAAASU1SbAAAA..."
                        },
                        "index": 0
                    }
                ]
            }
        }
    ],
    "usage": {
        "prompt_tokens": 6,
        "completion_tokens": 1368,
        "total_tokens": 1374
    }
}
```

### 响应参数对应关系

| 统一格式 | 原始格式 | 转换说明 |
|---------|---------|---------|
| `request_id` | `id` | 直接映射 |
| `output.choices[0].finish_reason` | `choices[0].finish_reason` | 直接映射 |
| `output.choices[0].message.role` | `choices[0].message.role` | 直接映射 |
| `output.choices[0].message.content[].image` | `choices[0].message.images[0].image_url.url` | 提取 Base64 图片数据 |
| `output.task_metric.SUCCEEDED` | - | 固定值 `1` |
| `output.task_metric.TOTAL` | - | 固定值 `1` |
| `output.task_metric.FAILED` | - | 固定值 `0` |
| `usage.image_count` | - | 根据 `images` 数组长度计算 |

---

# 统一错误格式

当任何模型返回错误时，插件会将其转换为统一的错误格式：

```json
{
    "code": "400",
    "message": {
        "error": "详细错误信息",
        "type": "invalid_request_error"
    },
    "data": "higress transformer plugin"
}
```

### 错误响应参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `code` | string | HTTP 状态码 |
| `message` | object/string | 原始错误信息（保留原始结构） |
| `data` | string | 固定值，标识来自 Higress Transformer 插件 |

### 响应头说明

插件会在响应头中添加以下信息：

| Header | 说明 | 示例值 |
|--------|------|--------|
| `model` | 使用的模型标识符 | `volcengine/volcengine/doubao-seedream-4.5` |
| `model_type` | 模型类型 | `t2i` |
| `x-transform-fail-stage` | 转换失败阶段（仅在失败时） | `request` / `response` |

---

## 使用示例

### cURL 示例

**重要**: 所有请求必须包含 `model` 和 `model_type` Header。

#### 豆包 SeeDream 4.5

```bash
curl -X POST http://172.22.221.212/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "model: volcengine/volcengine/doubao-seedream-4.5" \
  -H "model_type: t2i" \
  -d '{
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "input": {
      "messages": [
        {
          "role": "user",
          "content": [
            {
              "text": "生成一幅美丽的风景画"
            }
          ]
        }
      ]
    },
    "parameters": {
      "size": "1024*1024",
      "n": 1
    }
  }'
```

#### 通义千问图像生成

```bash
curl -X POST http://172.22.221.212/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "model: aliyun/aliyun/qwen-image-plus" \
  -H "model_type: t2i" \
  -d '{
    "model": "aliyun/aliyun/qwen-image-plus",
    "input": {
      "messages": [
        {
          "role": "user",
          "content": [
            {
              "text": "一副典雅庄重的对联"
            }
          ]
        }
      ]
    },
    "parameters": {
      "size": "1328*1328",
      "watermark": false
    }
  }'
```

#### Gemini 3 Image

```bash
curl -X POST http://172.22.221.212/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "model: openrouter/auto/gemini-3-image" \
  -H "model_type: t2i" \
  -d '{
    "model": "openrouter/auto/gemini-3-image",
    "input": {
      "messages": [
        {
          "role": "user",
          "content": [
            {
              "text": "Generate a beautiful sunset"
            }
          ]
        }
      ]
    },
    "extra_body": {
      "gemini_3_image": {
        "modalities": ["image"]
      }
    }
  }'
```

### Python 示例

```python
import requests

url = "http://172.22.221.212/v1/images/generations"

headers = {
    "Content-Type": "application/json",
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "model_type": "t2i"
}

payload = {
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "生成一幅美丽的风景画"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "size": "1024*1024",
        "n": 1
    }
}

response = requests.post(url, json=payload, headers=headers)
print(response.json())
```

### JavaScript 示例

```javascript
const url = "http://172.22.221.212/v1/images/generations";

const headers = {
    "Content-Type": "application/json",
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "model_type": "t2i"
};

const payload = {
    model: "volcengine/volcengine/doubao-seedream-4.5",
    input: {
        messages: [
            {
                role: "user",
                content: [
                    {
                        text: "生成一幅美丽的风景画"
                    }
                ]
            }
        ]
    },
    parameters: {
        size: "1024*1024",
        n: 1
    }
};

fetch(url, {
    method: "POST",
    headers: headers,
    body: JSON.stringify(payload)
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error("Error:", error));
```

---

## 注意事项

1. **必需的 HTTP Headers**: 所有请求必须在 HTTP Header 中包含 `model` 和 `model_type`，否则插件无法识别模型类型
2. **API 端点**: 统一使用 `http://172.22.221.212/v1/images/generations` 作为请求地址
3. **尺寸格式统一**: 所有模型的 `size` 参数统一使用 `"宽*高"` 格式（如 `"1328*1328"`），插件会自动转换为各模型的原生格式
4. **图片数量参数**: 统一使用 `parameters.n` 表示生成图片数量，插件会自动映射到各模型的对应参数
5. **响应格式**: 所有模型的响应都会转换为统一的 Qwen 格式，便于统一处理
6. **流式响应**: 豆包模型支持流式响应（`stream: true`），其他模型暂不支持
7. **图片有效期**: URL 格式的图片链接通常有时效性（24 小时），建议及时下载保存
8. **Base64 编码**: Gemini 返回 Base64 编码的图片，可直接用于前端显示或保存为文件
9. **错误处理**: 所有错误都会转换为统一的错误格式，包含 `code`、`message` 和 `data` 字段
10. **响应头信息**: 插件会在响应头中添加 `model`、`model_type` 和 `x-transform-fail-stage`（失败时）等信息
