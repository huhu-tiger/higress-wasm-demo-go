
# 文生图
[在线说明](https://www.volcengine.com/docs/82379/1541523?lang=zh)
## 模型: doubao-seedream-4.5:
### 请求参数:
```json
{
    "model": "volcengine/volcengine/doubao-seedream-4.5",
    "prompt": "生成一组共1张连贯插画，核心为同一庭院一角的四季变迁，以统一风格展现四季独特色彩、元素与氛围",
    "sequential_image_generation": "auto",
    "sequential_image_generation_options": {
        "max_images": 1
    },
    "response_format": "url",
    "size": "2K",
    "stream": false,
    "watermark": true
}
```
### 请求参数详解:

#### 顶层参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| model | string | 是 | 模型名称，固定为 `volcengine/volcengine/doubao-seedream-4.5` | `"volcengine/volcengine/doubao-seedream-4.5"` |
| prompt | string | 是 | 文本提示词，用于描述期望生成的图像内容。支持中英文，建议不超过300个汉字或600个英文单词 | `"生成一组共1张连贯插画，核心为同一庭院一角的四季变迁..."` |
| sequential_image_generation | string | 否 | 是否启用序列图片生成模式 | `"disabled"`（关闭）<br>`"auto"`（自动启用）<br>默认值：`"disabled"` |
| sequential_image_generation_options | object | 否 | 序列图片生成选项，当 `sequential_image_generation` 为 `"auto"` 时有效 | 见下方说明 |
| response_format | string | 否 | API 响应的格式 | `"url"`（返回图片链接）<br>`"b64_json"`（返回 Base64 编码）<br>默认值：`"url"` |
| size | string | 否 | 生成图像的尺寸。可以使用简化格式或直接指定宽高像素值 | 简化格式：`"2K"`、`"4K"`<br>像素格式：`"2048x2048"`<br>总像素范围：3686400 至 16777216<br>宽高比范围：1/16 至 16 |
| stream | boolean | 否 | 是否以流式方式返回生成的图片 | `true`（启用流式返回）<br>`false`（禁用流式返回）<br>默认值：`false` |
| watermark | boolean | 否 | 是否在生成的图片中添加水印 | `true`（添加水印）<br>`false`（不添加水印）<br>默认值：`true` |
| n | integer | 否 | 生成图片的数量（与 `sequential_image_generation_options.max_images` 功能相同，两者只需设置一个） | `1` 到 `15`（具体范围可能因配置而异） |
| seed | integer | 否 | 随机数种子，用于控制模型生成内容的随机性。使用相同的 `seed` 值可以使生成内容保持相对稳定 | `0` 到 `2147483647` |

#### sequential_image_generation_options 对象参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 | 取值范围/示例值 |
|:------:|:----:|:----:|:------:|:----:|:---------------:|
| max_images | integer | 否 | `15` | 指定生成的最大图片数量，仅在 `sequential_image_generation` 为 `"auto"` 时有效。**注意**: 此参数与顶层 `n` 参数功能相同，表示生成图片的数量，两者只需设置一个 | `1` 到 `15` |

**注意事项：**
- `doubao-seedream-4.5` 模型支持文生图功能
- `prompt` 参数是必填项，建议详细描述画面内容、风格、构图等
- `size` 参数支持简化格式（如 `2K`、`4K`）和像素格式（如 `2048x2048`）
- **`n` 和 `sequential_image_generation_options.max_images` 是同一个参数**，都表示生成图片的数量，两者只需设置一个。当 `sequential_image_generation` 设置为 `"auto"` 时，建议使用 `max_images`；否则使用 `n`
- `response_format` 为 `"url"` 时，返回的链接通常在生成后 24 小时内有效，建议及时下载保存
- `response_format` 为 `"b64_json"` 时，返回 Base64 编码字符串，适用于需要直接处理图片数据的场景
- `watermark` 默认值为 `true`，如需无水印图片请设置为 `false`
- `stream` 参数控制是否启用流式返回，流式返回适用于需要实时获取生成进度的场景

### transformer reqRules 配置

需要将统一格式转换为 doubao-seedream 格式，主要转换包括：

1. **提取 prompt**: 从 `input.messages[0].content[0].text` 提取为顶层 `prompt`
2. **提取 parameters**: 将 `parameters` 中的参数提升到顶层（除 `n` 外）
3. **映射 n 到 max_images**: 将 `parameters.n` 映射到 `sequential_image_generation_options.max_images`（注意：`n` 和 `max_images` 是同一个参数，表示生成图片的数量）
4. **提取 extra_body**: 将 `extra_body.doubao_seedream` 中的参数提升到顶层
5. **删除不需要的字段**: 删除 `input`、`parameters` 和 `extra_body` 对象

**转换前的统一格式请求**:
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



**转换后的 doubao-seedream 格式请求**:
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

**说明**: 
- 使用 GJSON Path 语法 `input.messages.0.content.0.text` 来访问嵌套数组中的字段
- `map` 操作会保留源字段，如果需要删除源字段，需要在最后使用 `remove` 操作
- **重要**: `parameters.n` 和 `sequential_image_generation_options.max_images` 是同一个参数，都表示生成图片的数量。在转换时，将 `parameters.n` 映射到 `sequential_image_generation_options.max_images`，不再保留 `n` 字段
- **size 参数统一处理**: 统一格式使用 qwen 方式（`"宽*高"`，如 `"1328*1328"`），在转换为 doubao 格式时会自动映射：
  - 如果总像素数接近 2K（约 4M 像素）且为正方形，映射到 `"2K"`
  - 如果总像素数接近 4K（约 16M 像素）且为正方形，映射到 `"4K"`
  - 其他情况转换为像素格式（`"宽x高"`，如 `"1328x1328"`）

### 响应参数:
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

或当 `response_format` 为 `"b64_json"` 时：

```json
{
    "model": "doubao-seedream-4-5-251128",
    "created": 1768313092,
    "data": [
        {
            "b64_json": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
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

### 响应参数详解:

#### 顶层响应参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| created | integer | 创建时间戳（Unix 时间戳） | `1677652288` |
| data | array | 生成结果数组，包含生成的图片信息 | `[{"url": "https://..."}]` 或 `[{"b64_json": "..."}]` |
| error | object | 可选，错误信息对象（仅在错误时返回） | 见下方说明 |

#### data 数组元素参数（当 response_format 为 "url" 时）

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| url | string | 生成图像的 URL 地址，通常在生成后 24 小时内有效 | `"https://cdn.modelgate.com/images/abc123.png"` |
| revised_prompt | string | 可选，修订后的提示词（如果 API 对提示词进行了优化） | - |

#### data 数组元素参数（当 response_format 为 "b64_json" 时）

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| b64_json | string | 生成图像的 Base64 编码字符串 | `"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="` |
| revised_prompt | string | 可选，修订后的提示词（如果 API 对提示词进行了优化） | - |

#### error 对象参数（错误时返回）

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| message | string | 错误信息描述 | `"Invalid prompt"` |
| type | string | 错误类型 | `"invalid_request_error"` |
| param | string | 可选，导致错误的参数名称 | `"prompt"` |
| code | string | 可选，错误码 | - |

**注意事项：**
- `created` 字段表示图片生成的时间戳（Unix 时间戳，秒级）
- `data` 数组的长度通常等于请求参数中的 `n` 值或 `sequential_image_generation_options.max_images` 值
- 当 `response_format` 为 `"url"` 时，返回的 URL 链接具有时效性（通常 24 小时内有效），需要及时下载保存
- 当 `response_format` 为 `"b64_json"` 时，返回 Base64 编码字符串，可以直接用于前端显示或保存
- `revised_prompt` 字段仅在 API 对原始提示词进行了优化或修订时返回
- 当请求失败时，响应中会包含 `error` 对象，用于描述错误信息
- 流式响应（`stream: true`）的格式可能与上述不同，具体格式请参考流式响应相关文档
### transformer respRules 配置（输出映射到 qwen-image 标准）

目标：将 doubao-seedream-4.5 的响应重写为与 qwen-image 相同的标准结构：

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
- **动态兼容性**: transformer 的 `map` 操作不支持动态遍历数组，需要为每个图片索引配置映射规则
- **多图处理**: 所有图片会被映射到 `output.choices.0.message.content` 数组中，索引从 0 开始
- **字段优先级**: `url` 和 `b64_json` 互斥，按配置顺序覆盖，如果 `data.0.url` 存在则使用它，否则使用 `data.0.b64_json`
- **安全性**: 
  - ✅ **不会失败**: 如果返回的图片数量少于映射规则配置的数量，不会失败
  - ✅ **自动跳过**: `map` 操作在 `fromKey` 不存在时会自动跳过（无操作），不会报错
  - ✅ **灵活配置**: 可以配置最大数量的映射规则，实际返回的图片数量可以少于配置数量
  - ⚠️ **超出限制**: 如果返回的图片数量超过配置的映射规则数量，超出部分不会被转换
- **配置建议**: 
  - 如果确定只返回 1 张图片，使用简化版配置
  - 如果需要支持多图，根据实际最大图片数量配置相应的映射规则（最多 15 张）
  - 可以安全地配置最大数量的映射规则，即使实际返回的图片数量较少也不会出错
- **doubao 响应特性**: 
  - `size` 为字符串（如 `2048x2048`），如需拆分为 `usage.height`/`usage.width`，需在上游先解析再写回响应体后再映射
  - 当 `response_format` 为 `b64_json` 时，`url` 字段不存在；当为 `url` 时，`b64_json` 不会返回
- **类型转换**: `created` 字段映射到 `request_id` 时使用 `value_type: string` 确保类型正确

