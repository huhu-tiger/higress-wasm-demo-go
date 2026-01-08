
# todo list
- [x] 更新makefile 参考ai-data-masking-go
- [x] 增加deploy_dev 目录，创建开发测试用的docker-compose.yml和envoy.yaml
- [x] 更新deploy_dev 中的 envoy.yaml使用 以下接口进行 openai接口的测试
    ```
    curl --location --request POST 'http://61.49.53.5:30002/v1/chat/completions' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "model": "DeepSeek-V3.2",
        "temperature": 0.1,
        "top_p": 1.0,
        "max_tokens": 2048,
        "echo": "False",
        "stream": true,
        "messages": [
            {
                "role": "user",
                "content": "生成1000字的童话故事"
            }
        ]
    }'
    ```
- [x] 更新deploy_dev 中的 envoy.yaml，增加对anthropic claude 接口的测试
```
curl --location --request POST 'http://220.181.114.184:30951/v1/messages' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer tk-OvOx9M2qhHxYHcO8SQJdAkFVHVnf1tUD' \
--data-raw '{
  "model": "z-ai/text/glm-4.5-air",
    "max_tokens": 5120,
  "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "hello"
          }
        ]
      }
    ],
    "stream":true
  
}'
```