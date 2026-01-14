

# 需求：多种请求格式与返回格式需要统一


# 实现逻辑
## 逻辑如下
读取请求头中 model 与 model_type ,例如model: volcengine/volcengine/doubao-seedream-4.5,model_type: t2i
在代码中编写struct，针对不同的 model与model_type实现不同的转换逻辑,实现统一请求，统一返回 结构体

## model_type 关系

t2i 代表文生图， i2i 代表图生图

# 待办事项

- [X] 读取transformer-vnet/文生图/doubao-seedream-4.5.md，实现请求与响应格式统一
- [X] 日志使用 transformer-vnet/wlog/log.go
- [X] 在transformer-vnet/bussiness/t2i/public.go定义 统一请求结构体 和统一返回结构体，在transformer-vnet/bussiness/t2i/doubao.go 定义doubao-seedream-4.5.md涉及的
      实际请求结构体与 实际返回结构体，在transformer-vnet/bussiness/t2i/doubao.go 定义 转换为统一请求返回结构体的方法，在main.go中引用逻辑
- [X] 如果转换失败，在返回头上添加失败的 阶段，请求或者响应阶段
- [X] 如果实际服务返回非正常状态，返回统一的错误提示，code: 来自实际返回状态码,message: 是实际错误返回结构体，data: "higress transformer plugin" 
- [X] 对请求的size参数进行统一处理，使用qwen的方式统一方式传入，豆包的2K,4K等参数 使用value映射
- [X] 增加对transformer-vnet/docs/文生图/qwen-image.md 统一请求与响应的支持，header的model与model_type参考Qwen文档
- [X] 加入 transformer-vnet/docs/文生图/gemini-3-image.md  统一请求与响应的支持，header的model与model_type参考Qwen文档