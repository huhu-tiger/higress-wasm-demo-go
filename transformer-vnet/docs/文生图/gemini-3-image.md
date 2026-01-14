
# 文生图
[在线说明](https://openrouter.ai/google/gemini-3-pro-image-preview/api)
## 模型: gemini-3-image:
### 请求参数:
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
    "provider":{
       "only":["Google"]
    }
}
```

### 请求参数详解:

#### 顶层参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| model | string | 是 | 模型名称，可通过 OpenRouter 使用 | `"openrouter/auto/gemini-3-image"` |
| messages | array | 是 | 消息数组，包含用户输入内容 | `[{"role": "user", "content": "..."}]` |
| modalities | array | 是 | 指定响应的输出模态类型 | `["image", "text"]` 或 `["image"]` |
| provider | object | 否 | 指定提供商过滤，限制使用特定的提供商 | `{"only": ["Google"]}` |

#### provider 对象参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| only | array | 否 | 限制只使用指定的提供商列表 | `["Google"]` |

#### messages 数组元素参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| role | string | 是 | 消息角色类型 | `"user"`（用户）<br>`"assistant"`（助手）<br>`"system"`（系统） |
| content | string/array | 是 | 消息内容。可以是字符串（纯文本）或数组（多模态内容，可包含文本和图片） | 字符串：`"Generate a beautiful sunset"`<br>数组：`[{"type": "text", "text": "..."}, {"type": "image_url", "image_url": {"url": "..."}}]` |

#### content 数组元素参数（多模态内容）

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| type | string | 是 | 内容类型 | `"text"`（文本）<br>`"image_url"`（图片 URL） |
| text | string | 否 | 文本内容（当 `type` 为 `"text"` 时） | `"Generate a beautiful sunset over mountains"` |
| image_url | object | 否 | 图片 URL 对象（当 `type` 为 `"image_url"` 时） | `{"url": "data:image/png;base64,..."}` 或 `{"url": "https://..."}` |

**注意事项：**
- `gemini-3-image` 模型支持文生图和图生图功能
- `modalities` 参数必须包含 `"image"` 才能生成图像
- `messages` 中的 `content` 可以是简单字符串（纯文本提示词）或数组（多模态内容，可包含参考图片）
- 支持混合最多 14 张参考图像来生成新图像
- 参考图片可以通过 `data:image/png;base64,...` 格式的 Base64 编码字符串或 URL 提供

### transformer reqRules 配置

需要将统一格式转换为 gemini-3-image 格式，主要转换包括：

1. **构建 messages 结构**: 
   - 提取 `input.messages[0].role` → `messages[0].role`
   - 提取 `input.messages[0].content[0].text` → `messages[0].content`（直接转换为字符串）
2. **提取 extra_body 中的参数到顶层**: 
   - 从 `extra_body.gemini_3_image.modalities` 提取到顶层 `modalities`
   - 从 `extra_body.gemini_3_image.provider` 提取到顶层 `provider`
3. **删除不需要的字段**: 
   - 删除 `input` 对象（已提取 messages）
   - 删除 `parameters` 对象（gemini-3-image 不需要）
   - 删除 `extra_body` 对象（已提取 modalities 和 provider，删除整个对象及其所有嵌套内容）

**转换前的统一格式请求**:
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
              "provider":{
                "only":["Google"]
                }
        }
    }
}
```



**转换后的 gemini-3-image 格式请求**:
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
      "provider":{
       "only":["Google"]
    }
}
```

**说明**: 
- 直接从 `input.messages.0.content.0.text` 提取文本值，并映射到 `messages.0.content`（字符串格式）
- 同时提取 `input.messages.0.role` 到 `messages.0.role`，构建完整的 messages 结构
- 从 `extra_body.gemini_3_image.modalities` 提取到顶层 `modalities`，确保参数被提取到上一层
- 从 `extra_body.gemini_3_image.provider` 提取到顶层 `provider`，用于指定提供商过滤
- 使用 `map` 操作可以直接将嵌套的文本值提取为字符串，无需先映射整个 messages 数组再转换
- **重要**: 删除操作必须在所有提取操作之后执行，确保需要的参数已经提取到顶层
- 删除 `extra_body` 会删除整个 `extra_body` 对象及其所有嵌套内容（包括 `extra_body.gemini_3_image` 等）
- 如果 `content` 已经是字符串格式，可以简化配置，直接映射整个 `input.messages` 到 `messages`

### 响应参数:
```json
{
    "id": "gen-1768313027-nwX1nWNYMC56FXn2jLuT",
    "provider": "Google",
    "model": "google/gemini-3-pro-image-preview",
    "object": "chat.completion",
    "created": 1768313028,
    "choices": [
        {
            "logprobs": null,
            "finish_reason": "stop",
            "native_finish_reason": "STOP",
            "index": 0,
            "message": {
                "role": "assistant",
                "content": "",
                "refusal": null,
                "reasoning": "data",
                "reasoning_details": [
                    {
                        "format": "google-gemini-v1",
                        "index": 0,
                        "type": "reasoning.text",
                        "text": "**Imagining a Sunset Scene**\n\nI am currently focusing on crafting a sunset scene over a mountain range. The goal is to incorporate specific elements of light, color, and landscape features. The aim is to make the image visually compelling.\n\n\n**Composing the Elements**\n\nI'm now structuring the scene, focusing on the interplay of light and landscape. The rugged peaks will be silhouetted against the sunset colors. I'm adding a reflective alpine lake in the foreground, with details like a winding trail and wildflowers to enhance the depth. I am refining the atmospheric details to include mist and clouds catching the light.\n\n\n**Considering Visual Fidelity**\n\nI'm currently focused on evaluating the image against the provided description. The goal is to determine how accurately the scene has been rendered, specifically the arrangement and characteristics of the sunset and mountain elements. My next step will be to assess how well the image delivers the emotional impact implied by the prompt.\n\n\n**Analyzing Prompt Alignment**\n\nI've checked the generated image. My assessment is that the rendering largely adheres to the initial request. I've noted the fidelity in the sunset and mountain details. The next step is a review of the image's overall consistency with the emotional intent described in the initial text.\n\n\n"
                    },
                    {
                        "format": "google-gemini-v1",
                        "index": 0,
                        "type": "reasoning.encrypted",
                        "data": "Cuy/egGPPWtfnY6sIvJ3HiiXxUP0Oa9JLV1seuEYDaxxG3TzLfTRoxNxIesI7goTjQ2AsEx2epBB3vCtUKTv3M8P+dqTHbZpR7Z8J1/nOMyBm+brSFkw8+BpqMJcyQQg58jhOL96LvnsUQl5aAptIiXUoUSoACLxGtrPWF6QX65TykNlnM0pNlR/p1fgSfi5lLbKHWe1q/d1KftoQCFYFgQ/bayxr6TfGJ4L8tt9jRaut+6sjGkAMz3v7HUPB9SAeo48CEEFWjn36Z/GmDCI/6aVFQHFA/+r//Xao2DKqyfuIyCIIVkdz7TYuXC1L+6rJ2YZyIrtL49kEa1Xzo0krhFf/872AYqrUxAi9a3IccSbG6Q7srstMDJpk9/X+o7ahUF9v6sZsqoJ7lX7UeRd7LG5xwT6pQ40fSAOHShJa8NjEu8gbkz6zh/HF4m6O8xlruWBqGjoxaslk7w/UNIPmYQnD3ax/xai/pIWBlf5yE3BitqLxBIhI"
                    }
                    ],
                    "annotations": [],
                    "images": [
                        {
                        "type": "image_url",
                        "image_url": {
                             "url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAABYAAAAMACAIAAAASU1SbAAAAiXpUWHRSYXcgcHJvZmlsZSB0eXBlIGlwdGMAAAiZTYwxDgIxDAT7vOKekDjrtV1T0VHwgbtcIiEhgfh/"
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
        "total_tokens": 1374,
        "cost": 0.137388,
        "is_byok": false,
        "prompt_tokens_details": {
            "cached_tokens": 0,
            "audio_tokens": 0,
            "video_tokens": 0
        },
        "cost_details": {
            "upstream_inference_cost": null,
            "upstream_inference_prompt_cost": 0.000012,
            "upstream_inference_completions_cost": 0.137376
        },
        "completion_tokens_details": {
            "reasoning_tokens": 248,
            "image_tokens": 1120
        }
    }
}
```

### 响应参数详解:

#### 顶层响应参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| id | string | 请求唯一标识符 | `"gen-abc123"` |
| model | string | 使用的模型名称 | `"openrouter/auto/gemini-3-image"` |
| created | integer | 创建时间戳（Unix 时间戳） | `1677652288` |
| choices | array | 生成结果选择数组 | `[{"index": 0, "message": {...}, "finish_reason": "stop"}]` |
| usage | object | Token 使用情况统计对象 | 见下方说明 |
| error | object | 可选，错误信息对象（仅在错误时返回） | 见下方说明 |

#### choices 数组元素参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| index | integer | 当前选择项的索引位置 | `0` |
| message | object | 消息对象，包含生成的内容 | 见下方说明 |
| finish_reason | string | 完成原因，指示模型停止生成的原因 | 可能的值：<br>- `"stop"`：模型自然停止生成（正常完成）<br>- `"length"`：达到上下文长度或 `max_tokens` 限制<br>- `"content_filter"`：输出内容触发过滤策略<br>- `"tool_calls"`：模型决定调用外部工具 |

#### message 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| role | string | 角色类型，固定为 `"assistant"` | `"assistant"` |
| content | array | 内容数组，包含生成的文本和图片 | `[{"type": "text", "text": "..."}, {"type": "image_url", "image_url": {...}}]` |

#### content 数组元素参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| type | string | 内容类型标识 | `"text"`（文本内容）<br>`"image_url"`（图片内容） |
| text | string | 文本内容（当 `type` 为 `"text"` 时） | `"I've generated a beautiful sunset image for you."` |
| image_url | object | 图片 URL 对象（当 `type` 为 `"image_url"` 时） | `{"url": "data:image/png;base64,..."}` |

#### image_url 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| url | string | 生成图像的 URL，通常为 Base64 编码的 data URL | `"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="` |

#### usage 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| prompt_tokens | integer | 提示词 token 数量 | `10` |
| completion_tokens | integer | 完成内容 token 数量 | `5` |
| total_tokens | integer | 总 token 数量 | `15` |

#### error 对象参数（错误时返回）

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| message | string | 错误信息描述 | `"Invalid request"` |
| type | string | 错误类型 | `"invalid_request_error"` |
| param | string | 可选，导致错误的参数名称 | `"modalities"` |
| code | string | 可选，错误码 | - |

**注意事项：**
- `id` 字段是请求的唯一标识符，可用于追踪和调试
- `created` 字段表示响应生成的时间戳（Unix 时间戳，秒级）
- `choices` 数组通常包含一个元素，但在某些配置下可能包含多个候选结果
- `content` 数组可能同时包含文本和图片内容，需要根据 `type` 字段进行区分处理
- `image_url.url` 字段包含 Base64 编码的图片数据，格式为 `data:image/png;base64,...` 或 `data:image/jpeg;base64,...`
- Base64 编码的图片数据可以直接用于前端显示，也可以解码后保存为文件
- `finish_reason` 为 `"stop"` 表示正常完成，其他值可能表示异常情况或特殊状态
- `usage` 对象提供 token 使用统计，可用于计费和监控
- 当请求失败时，响应中会包含 `error` 对象，用于描述错误信息
- 流式响应（`stream: true`）的格式可能与上述不同，具体格式请参考流式响应相关文档
### transformer respRules 配置（输出映射到 qwen-image 标准）

目标：将 gemini-3-image 的响应重写为与 qwen-image 相同的标准结构：

```json
{
        "request_id": "...",
        "output": {
                "choices": [
                        {
                                "finish_reason": "stop",
                                "message": {
                                        "role": "assistant",
                                        "content": [
                                                {"text": "..."},
                                                {"image": "https://..."}
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

**说明**
- content 中的文本和图片索引仅作示例，如返回的图片在 content 中的位置不同，调整 `choices.0.message.content.X` 的索引即可。
- 如果响应包含多张图片，可继续按索引添加映射条目以填充更多 `content` 元素或 `choices` 条目。
- gemini 响应不提供分辨率信息，若需要 `usage.height`/`usage.width`，可在上游推理服务补充后再映射。

