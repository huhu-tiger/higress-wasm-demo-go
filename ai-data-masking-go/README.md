# AI 数据脱敏插件 (Golang 版本)

这是 AI 数据脱敏插件的 Golang 实现版本，功能与 Rust 版本相同。

## 功能说明

对请求/返回中的敏感词拦截、替换

### 处理数据范围
- openai协议：请求/返回对话内容
- jsonpath：只处理指定字段
- raw：整个请求/返回body

### 敏感词拦截
- 处理数据范围中出现敏感词直接拦截，返回预设错误信息
- 支持系统内置敏感词库和自定义敏感词

### 敏感词替换
- 将请求数据中出现的敏感词替换为脱敏字符串，传递给后端服务。可保证敏感数据不出域
- 部分脱敏数据在后端服务返回后可进行还原
- 自定义规则支持标准正则和grok规则，替换字符串支持变量替换

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`991`

## 配置字段

| 名称 | 数据类型 | 默认值 | 描述 |
| -------- | --------  | -------- | -------- |
| deny_openai | bool | true | 对openai协议进行拦截 |
| deny_jsonpath | string | [] | 对指定jsonpath拦截 |
| deny_raw | bool | false | 对原始body拦截 |
| system_deny | bool | false | 开启内置拦截规则 |
| deny_code | int | 200 | 拦截时http状态码 |
| deny_message | string | 提问或回答中包含敏感词，已被屏蔽 | 拦截时ai返回消息 |
| deny_raw_message | string | {"errmsg":"提问或回答中包含敏感词，已被屏蔽"} | 非openai拦截时返回内容 |
| deny_content_type | string | application/json | 非openai拦截时返回content_type头 |
| deny_words | array of string | [] | 自定义敏感词列表 |
| replace_roles | array | - | 自定义敏感词正则替换 |
| replace_roles.regex | string | - | 规则正则(内置GROK规则) |
| replace_roles.type | [replace, hash] | - | 替换类型 |
| replace_roles.restore | bool | false | 是否恢复 |
| replace_roles.value | string | - | 替换值（支持正则变量） |

## 配置示例

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-data-masking
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  defaultConfig:
    system_deny: true
    deny_openai: true
    deny_jsonpath:
      - "$.messages[*].content"
    deny_raw: true
    deny_code: 200
    deny_message: "提问或回答中包含敏感词，已被屏蔽"
    deny_raw_message: "{\"errmsg\":\"提问或回答中包含敏感词，已被屏蔽\"}"
    deny_content_type: "application/json"
    deny_words: 
      - "自定义敏感词1"
      - "自定义敏感词2"
    replace_roles:
      - regex: "%{MOBILE}"
        type: "replace"
        value: "****"
      - regex: "%{EMAILLOCALPART}@%{HOSTNAME:domain}"
        type: "replace"
        restore: true
        value: "****@$domain"
      - regex: "%{IP}"
        type: "replace"
        restore: true
        value: "***.***.***.***"
      - regex: "%{IDCARD}"
        type: "replace"
        value: "****"
      - regex: "sk-[0-9a-zA-Z]*"
        restore: true
        type: "hash"
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-data-masking:1.0.0
  phase: AUTHN
  priority: 991
```

## 构建

```bash
cd /data/work/higress/plugins/wasm-go
PLUGIN_NAME=ai-data-masking make build
```

## 相关说明

- 流模式中如果脱敏后的词被多个chunk拆分，可能无法进行还原
- 流模式中，如果敏感词语被多个chunk拆分，可能会有敏感词的一部分返回给用户的情况
- grok 内置规则列表 https://help.aliyun.com/zh/sls/user-guide/grok-patterns
- 内置敏感词库数据来源 https://github.com/houbb/sensitive-word-data/tree/main/src/main/resources
- 由于敏感词列表是在文本分词后进行匹配的，所以请将 `deny_words` 设置为单个单词，英文多单词情况如 `hello word` 可能无法匹配

## 与 Rust 版本的差异

1. **敏感词检测**：当前使用简单的字符串包含匹配，Rust 版本使用 jieba 分词
2. **GROK 支持**：当前是简化实现，支持常见模式
3. **流式响应处理**：基础实现，需要进一步完善 SSE 解析
4. **系统敏感词库**：需要从资源文件加载

## 待完善功能

- [ ] 完整的 GROK 模式支持
- [ ] 系统敏感词库从资源文件加载
- [ ] 更完善的流式响应（SSE）解析
- [ ] 使用分词库进行敏感词检测（如 gojieba）

