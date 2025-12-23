from fastapi import FastAPI, Request, Response, Query, Path, Body, Header
from fastapi.responses import JSONResponse, RedirectResponse, StreamingResponse
from typing import Optional, Dict, Any
import asyncio
import json
import gzip
import zlib
from datetime import datetime
import uvicorn
import os
from pathlib import Path as PathLib
app = FastAPI() 

def get_client_ip(request: Request) -> str:
    """获取客户端真实IP地址"""
    # 优先从 X-Forwarded-For 获取（代理场景）
    forwarded = request.headers.get("X-Forwarded-For")
    if forwarded:
        return forwarded.split(",")[0].strip()
    # 其次从 X-Real-IP 获取
    real_ip = request.headers.get("X-Real-IP")
    if real_ip:
        return real_ip
    # 最后从客户端连接获取
    return request.client.host if request.client else "unknown"

@app.post("/v1/chat/completions")
async def echo_chat_completions(
    request: Request,
    body: Any = Body(None)
):
    """返回一个固定的 OpenAI Chat Completions 格式响应"""
    # 检查是否为流式请求
    try:
        body_data = await request.json()
        is_stream = body_data.get("stream", False) if isinstance(body_data, dict) else False
    except:
        is_stream = False
    
    # 如果是流式请求，返回 stream.txt 中的数据
    if is_stream:
        # 获取 stream.txt 文件路径（与 echo.py 同目录）
        current_dir = PathLib(__file__).parent
        stream_file = current_dir / "stream.txt"
        
        async def generate_stream():
            """异步生成器，逐行读取 stream.txt 并返回"""
            if stream_file.exists():
                with open(stream_file, 'r', encoding='utf-8') as f:
                    for line in f:
                        # 确保每行以换行符结尾
                        if line and not line.endswith('\n'):
                            yield line + '\n'
                        else:
                            yield line
                        # 添加小延迟以模拟真实流式响应
                        await asyncio.sleep(0.01)
            else:
                # 如果文件不存在，返回错误信息
                error_msg = "data: {\"error\": \"stream.txt file not found\"}\n\n"
                yield error_msg
        
        return StreamingResponse(
            generate_stream(),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "Connection": "keep-alive",
            }
        )
    
    # 非流式请求，返回原来的 JSON 响应
    now_ts = int(datetime.utcnow().timestamp())
    return {
        "id": "chatcmpl-fixed-example",
        "object": "chat.completion",
        "created": now_ts,
        "model": "demo-model",
        "choices": [
            {
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": "这是一个固定的测试回复,测试敏感词，用于验证 AI 数据脱敏插件是否生效。",
                    "reasoning": "这是一个固定的测试回复，用于验证 AI 数据脱敏插件是否生效。"
                },
                "finish_reason": "stop"
            }
        ],
        "usage": {
            "prompt_tokens": 10,
            "completion_tokens": 20,
            "total_tokens": 30
        }
    }

@app.get("/get")
async def echo_get(
    request: Request,
    q: Optional[str] = Query(None, description="查询参数示例")
):
    """返回GET请求的所有信息"""
    return {
        "args": dict(request.query_params),
        "headers": dict(request.headers),
        "origin": get_client_ip(request),
        "url": str(request.url),
        "method": request.method
    }


@app.post("/posts")
async def echo_post(
    request: Request,
    body: Any = Body(None)
):
    """返回POST请求的所有信息"""
    try:
        body_data = await request.json()
    except:
        try:
            body_data = (await request.body()).decode('utf-8')
        except:
            body_data = None
    
    return {
        "args": dict(request.query_params),
        "data": body_data,
        "files": {},
        "form": {},
        "headers": dict(request.headers),
        "json": body_data if isinstance(body_data, (dict, list)) else None,
        "origin": get_client_ip(request),
        "url": str(request.url),
        "method": request.method
    }


@app.put("/put")
async def echo_put(
    request: Request,
    body: Any = Body(None)
):
    """返回PUT请求的所有信息"""
    try:
        body_data = await request.json()
    except:
        try:
            body_data = (await request.body()).decode('utf-8')
        except:
            body_data = None
    
    return {
        "args": dict(request.query_params),
        "data": body_data,
        "files": {},
        "form": {},
        "headers": dict(request.headers),
        "json": body_data if isinstance(body_data, (dict, list)) else None,
        "origin": get_client_ip(request),
        "url": str(request.url),
        "method": request.method
    }


@app.patch("/patch")
async def echo_patch(
    request: Request,
    body: Any = Body(None)
):
    """返回PATCH请求的所有信息"""
    try:
        body_data = await request.json()
    except:
        try:
            body_data = (await request.body()).decode('utf-8')
        except:
            body_data = None
    
    return {
        "args": dict(request.query_params),
        "data": body_data,
        "files": {},
        "form": {},
        "headers": dict(request.headers),
        "json": body_data if isinstance(body_data, (dict, list)) else None,
        "origin": get_client_ip(request),
        "url": str(request.url),
        "method": request.method
    }


@app.delete("/delete")
async def echo_delete(request: Request):
    """返回DELETE请求的所有信息"""
    return {
        "args": dict(request.query_params),
        "headers": dict(request.headers),
        "origin": get_client_ip(request),
        "url": str(request.url),
        "method": request.method
    }


@app.get("/status/{code}")
async def echo_status(
    code: int = Path(..., description="HTTP状态码", ge=100, le=599)
):
    """返回指定的HTTP状态码"""
    return JSONResponse(
        status_code=code,
        content={
            "status": code,
            "message": f"返回状态码 {code}"
        }
    )


@app.get("/headers")
async def echo_headers(request: Request):
    """返回请求头信息"""
    return {
        "headers": dict(request.headers)
    }


@app.get("/ip")
async def echo_ip(request: Request):
    """返回客户端IP地址"""
    return {
        "origin": get_client_ip(request)
    }


@app.get("/user-agent")
async def echo_user_agent(request: Request):
    """返回User-Agent信息"""
    return {
        "user-agent": request.headers.get("User-Agent", "unknown")
    }


@app.get("/delay/{seconds}")
async def echo_delay(
    seconds: float = Path(..., description="延迟秒数", ge=0, le=10)
):
    """延迟指定秒数后返回响应"""
    await asyncio.sleep(seconds)
    return {
        "delay": seconds,
        "message": f"延迟 {seconds} 秒后返回"
    }


@app.get("/json")
async def echo_json():
    """返回JSON格式数据"""
    return {
        "slideshow": {
            "author": "Yours Truly",
            "date": "date of publication",
            "slides": [
                {
                    "title": "Wake up to WonderWidgets!",
                    "type": "all"
                },
                {
                    "title": "Overview",
                    "type": "all"
                }
            ],
            "title": "Sample Slide Show"
        }
    }


@app.get("/cookies")
async def echo_cookies(request: Request):
    """返回cookies信息"""
    return {
        "cookies": dict(request.cookies)
    }


@app.get("/cookies/set")
async def echo_cookies_set(
    request: Request,
    name: str = Query(..., description="Cookie名称"),
    value: str = Query(..., description="Cookie值")
):
    """设置cookie并重定向"""
    response = RedirectResponse(url="/echo/cookies")
    response.set_cookie(key=name, value=value)
    return response


@app.get("/redirect/{n}")
async def echo_redirect(
    n: int = Path(..., description="重定向次数", ge=1, le=5)
):
    """执行n次重定向"""
    if n == 1:
        return RedirectResponse(url="/echo/get")
    else:
        return RedirectResponse(url=f"/echo/redirect/{n-1}")


@app.get("/redirect-to")
async def echo_redirect_to(
    url: str = Query(..., description="重定向目标URL")
):
    """重定向到指定URL"""
    return RedirectResponse(url=url)


@app.get("/gzip")
async def echo_gzip(request: Request):
    """返回gzip压缩的响应"""
    data = {
        "gzipped": True,
        "method": request.method,
        "origin": get_client_ip(request)
    }
    json_str = json.dumps(data, ensure_ascii=False)
    compressed = gzip.compress(json_str.encode('utf-8'))
    
    return Response(
        content=compressed,
        media_type="application/json",
        headers={
            "Content-Encoding": "gzip",
            "Content-Length": str(len(compressed))
        }
    )


@app.get("/deflate")
async def echo_deflate(request: Request):
    """返回deflate压缩的响应"""
    data = {
        "deflated": True,
        "method": request.method,
        "origin": get_client_ip(request)
    }
    json_str = json.dumps(data, ensure_ascii=False)
    compressed = zlib.compress(json_str.encode('utf-8'))
    
    return Response(
        content=compressed,
        media_type="application/json",
        headers={
            "Content-Encoding": "deflate",
            "Content-Length": str(len(compressed))
        }
    )


@app.get("/uuid")
async def echo_uuid():
    """返回UUID"""
    import uuid
    return {
        "uuid": str(uuid.uuid4())
    }


@app.get("/base64/{value}")
async def echo_base64(
    value: str = Path(..., description="Base64编码的字符串")
):
    """解码Base64字符串"""
    import base64
    try:
        decoded = base64.b64decode(value).decode('utf-8')
        return {
            "base64": value,
            "decoded": decoded
        }
    except Exception as e:
        return {
            "error": f"解码失败: {str(e)}"
        }


@app.get("/bytes/{n}")
async def echo_bytes(
    n: int = Path(..., description="字节数", ge=1, le=10000)
):
    """返回指定数量的随机字节"""
    import random
    import string
    random_bytes = ''.join(random.choices(string.ascii_letters + string.digits, k=n))
    return Response(
        content=random_bytes.encode('utf-8'),
        media_type="application/octet-stream"
    )


@app.get("/stream/{n}")
async def echo_stream(
    n: int = Path(..., description="流数据行数", ge=1, le=100)
):
    """返回流式数据"""
    async def generate():
        for i in range(n):
            data = {
                "id": i,
                "random": hash(f"{i}_{datetime.now()}") % 10000
            }
            yield json.dumps(data, ensure_ascii=False) + "\n"
            await asyncio.sleep(0.1)  # 模拟流式输出
    
    return Response(
        content=generate(),
        media_type="application/x-ndjson"
    )


@app.get("/")
async def echo_index():
    """Echo服务首页，列出所有可用端点"""
    return {
        "message": "Echo/Httpbin 服务",
        "endpoints": {
            "GET /echo/get": "返回GET请求信息",
            "POST /echo/post": "返回POST请求信息",
            "PUT /echo/put": "返回PUT请求信息",
            "PATCH /echo/patch": "返回PATCH请求信息",
            "DELETE /echo/delete": "返回DELETE请求信息",
            "GET /echo/status/{code}": "返回指定HTTP状态码",
            "GET /echo/headers": "返回请求头",
            "GET /echo/ip": "返回客户端IP",
            "GET /echo/user-agent": "返回User-Agent",
            "GET /echo/delay/{seconds}": "延迟响应",
            "GET /echo/json": "返回JSON数据",
            "GET /echo/cookies": "返回cookies",
            "GET /echo/cookies/set": "设置cookie",
            "GET /echo/redirect/{n}": "重定向n次",
            "GET /echo/redirect-to": "重定向到指定URL",
            "GET /echo/gzip": "返回gzip压缩响应",
            "GET /echo/deflate": "返回deflate压缩响应",
            "GET /echo/uuid": "返回UUID",
            "GET /echo/base64/{value}": "解码Base64",
            "GET /echo/bytes/{n}": "返回n字节随机数据",
            "GET /echo/stream/{n}": "返回流式数据"
        }
    }

if __name__ == "__main__":
    uvicorn.run(app='echo:app', host="0.0.0.0", port=8000, reload=True)