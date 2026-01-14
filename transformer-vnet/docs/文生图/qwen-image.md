
# 文生图
[在线说明](https://bailian.console.aliyun.com/cn-beijing/?spm=5176.29597918.J_tAwMEW-mKC1CPxlfy227s.1.7bd97b08Cz7wSh&tab=api#/api/?type=model&url=2975126)
## 模型: qwen-image-plus:
### 请求参数:
```json
{
    "model": "aliyun/aliyun/qwen-image-plus",
    "input": {
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "text": "一副典雅庄重的对联悬挂于厅堂之中，房间是个安静古典的中式布置，桌子上放着一些青花瓷，对联上左书“义本生知人机同道善思新”，右书“通云赋智乾坤启数高志远”， 横批“智启通义”，字体飘逸，在中间挂着一幅中国风的画作，内容是岳阳楼。"
                    }
                ]
            }
        ]
    },
    "parameters": {
        "negative_prompt": "",
        "prompt_extend": true,
        "watermark": false,
        "size": "1328*1328"
    }
}
```
### 请求参数详解:

#### 顶层参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| model | string | 是 | 模型名称，固定为 `aliyun/aliyun/qwen-image-plus` | `"aliyun/aliyun/qwen-image-plus"` |
| input | object | 是 | 输入内容对象 | 见下方说明 |
| parameters | object | 是 | 生成参数对象 | 见下方说明 |

#### input 对象参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| messages | array | 是 | 消息数组，包含用户输入 | `[{"role": "user", "content": [...]}]` |

#### messages 数组元素参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| role | string | 是 | 角色类型，固定为 `"user"` | `"user"` |
| content | array | 是 | 内容数组，包含文本或图片 | `[{"text": "..."}]` |

#### content 数组元素参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|:------:|:----:|:----:|:----:|:-----:|
| text | string | 是 | 文本提示词，用于描述期望生成的图像内容。支持中英文，建议详细描述画面内容、风格、构图等 | `"一副典雅庄重的对联悬挂于厅堂之中..."` |

#### parameters 对象参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 | 取值范围/示例值 |
|:------:|:----:|:----:|:------:|:----:|:---------------:|
| size | string | 否 | - | 生成图像的分辨率，格式为 `"宽*高"`。仅当 `n=1` 时支持设置 | 支持的固定尺寸：<br>- `"1664*928"` (16:9)<br>- `"928*1664"` (9:16)<br>- `"1328*1328"` (1:1)<br>- `"1472*1104"` (4:3)<br>- `"1104*1472"` (3:4) |
| negative_prompt | string | 否 | `""` | 反向提示词，描述不希望在生成图像中出现的内容。支持中英文，长度上限为 500 个字符，超出部分会自动截断 | 示例：`"低质量, 模糊, 多余手指, 残缺, 错误比例"` |
| prompt_extend | boolean | 否 | `true` | 是否开启提示词智能扩展。开启后，模型会对正向提示词进行智能改写，以提升生成效果，但不会修改反向提示词 | `true` / `false` |
| watermark | boolean | 否 | `false` | 是否在生成的图像右下角添加 "Qwen-Image" 水印 | `true` / `false` |
| n | integer | 否 | `1` | 生成图像的数量 | `1` 到 `6` |
| seed | integer | 否 | - | 随机数种子，用于控制模型生成内容的随机性。使用相同的 `seed` 值可以使生成内容保持相对稳定。如果未提供，算法将自动生成一个随机数作为种子 | `0` 到 `2147483647` |
| steps | integer | 否 | `20` | 推理步数，控制生成图像的精细度。步数越多，生成质量可能越高，但耗时也越长 | `1` 到 `50`，推荐值：`20` |
| num_inference_steps | integer | 否 | `20` | 推理步数（与 `steps` 参数功能相同，不同 API 版本可能使用不同字段名） | `1` 到 `50`，推荐值：`20` |
| guidance_scale | float | 否 | `3.5` | 引导强度，决定图像对提示词的遵循程度。值越大，图像越遵循提示词；值越小，图像越有艺术自由度 | `1.0` 到 `10.0`，推荐范围：`4.0` 到 `5.0`，默认：`3.5` |
| enable_safety_checker | boolean | 否 | `true` | 是否启用安全检查器，用于过滤不当内容 | `true` / `false` |
| output_format | string | 否 | `"png"` | 输出图像的格式 | `"png"` / `"jpg"` / `"jpeg"` |
| acceleration | string | 否 | - | 加速选项，用于控制生成速度 | `"none"` 或其他可选值（需参考最新文档） |

**注意事项：**
- `qwen-image-plus` 模型仅支持文生图功能，不支持图生图
- `size` 参数必须使用支持的固定尺寸之一，且仅当 `n=1` 时支持设置
- `negative_prompt` 如果为空字符串，表示不使用反向提示词
- `prompt_extend` 建议保持为 `true` 以获得更好的生成效果
- `n` 参数控制生成图像数量，范围 1-6 张
- `seed` 参数可用于复现相同提示词的生成结果
- `steps` 和 `num_inference_steps` 功能相同，不同 API 版本可能使用不同字段名，建议优先使用 `steps`
- `guidance_scale` 推荐值在 4.0-5.0 之间，可在提示一致性和艺术自由度之间取得平衡
- 部分参数（如 `enable_safety_checker`、`output_format`、`acceleration`）可能因 API 版本而异，建议参考最新官方文档

### transformer reqRules 配置

qwen-image 使用主格式，无需转换，`reqRules` 可以为空或仅包含其他通用规则。

**转换前的统一格式请求**:
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


**转换后的请求**（与转换前相同，无需转换）:
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

### 响应参数:
```json
{
    "output": {
        "choices": [
            {
                "finish_reason": "stop",
                "message": {
                    "content": [
                        {
                            "image": "https://dashscope-result-wlcb-acdr-1.oss-cn-wulanchabu-acdr-1.aliyuncs.com/7d/6a/20260113/cfc32567/8cbed912-b740-4786-b770-ddf664508c242990979895.png?Expires=1768898836&OSSAccessKeyId=LTAI5tKPD3TMqf2Lna1fASuh&Signature=hbYdZ%2B%2FJ5vBBjju7M5FoigeBWs8%3D"
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

### 响应参数详解:

#### 顶层响应参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| output | object | 输出结果对象 | 见下方说明 |
| usage | object | 使用情况统计对象 | 见下方说明 |
| request_id | string | 请求唯一标识符 | `"8cbed912-b740-4786-b770-ddf664508c24"` |
| code | string | 可选，错误码（仅在错误时返回） | - |
| message | string | 可选，错误信息（仅在错误时返回） | - |
| status_code | integer | 可选，HTTP 状态码 | `200` |

#### output 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| choices | array | 生成结果选择数组 | `[{"finish_reason": "stop", "message": {...}}]` |
| task_metric | object | 任务执行统计信息 | 见下方说明 |

#### choices 数组元素参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| finish_reason | string | 完成原因，指示模型停止生成的原因 | 可能的值：<br>- `"stop"`：模型自然停止生成（正常完成）<br>- `"length"`：达到上下文长度或 `max_tokens` 限制<br>- `"content_filter"`：输出内容触发过滤策略<br>- `"tool_calls"`：模型决定调用外部工具<br>- `"insufficient_system_resource"`：系统推理资源不足 |
| message | object | 消息对象，包含生成的内容 | 见下方说明 |
| index | integer | 可选，当前选择项的索引位置 | `0` |

#### message 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| role | string | 角色类型，固定为 `"assistant"` | `"assistant"` |
| content | array | 内容数组，包含生成的图片 | `[{"image": "https://..."}]` |

#### content 数组元素参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| image | string | 生成图像的 URL 地址，包含 OSS 签名信息，具有时效性 | `"https://dashscope-result-...oss-cn-wulanchabu-acdr-1.aliyuncs.com/..."` |
| type | string | 可选，内容类型标识，通常为 `"image"` | `"image"` |
| text | string | 可选，文本内容（如果响应中包含文本描述） | - |

#### task_metric 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| FAILED | integer | 失败的任务数量 | `0` |
| SUCCEEDED | integer | 成功的任务数量 | `1` |
| TOTAL | integer | 总任务数量 | `1` |

#### usage 对象参数

| 参数名 | 类型 | 说明 | 示例值 |
|:------:|:----:|:----:|:-----:|
| height | integer | 生成图像的高度（像素） | `1328` |
| width | integer | 生成图像的宽度（像素） | `1328` |
| image_count | integer | 生成图像的数量 | `1` |
| total_tokens | integer | 可选，总 token 数量（如果 API 支持 token 统计） | - |
| prompt_tokens | integer | 可选，提示词 token 数量（如果 API 支持 token 统计） | - |
| completion_tokens | integer | 可选，完成内容 token 数量（如果 API 支持 token 统计） | - |

**注意事项：**
- `image` URL 包含 OSS 签名参数（`Expires`、`OSSAccessKeyId`、`Signature`），具有时效性，需要及时保存
- `finish_reason` 为 `"stop"` 表示正常完成，其他值可能表示异常情况或特殊状态
- `task_metric` 中的 `SUCCEEDED` 和 `TOTAL` 通常相等，表示所有任务都成功完成
- 当请求失败时，响应中可能包含 `code` 和 `message` 字段，用于描述错误信息
- `usage` 中的 token 统计字段（`total_tokens`、`prompt_tokens`、`completion_tokens`）可能因 API 版本而异，不是所有版本都支持
- `choices` 数组的长度通常等于请求参数中的 `n` 值（生成图像的数量）