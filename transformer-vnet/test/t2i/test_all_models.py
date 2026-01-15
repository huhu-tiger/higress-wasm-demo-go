#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Transformer-VNet T2I 模型集成测试脚本
自动切换 Envoy 配置并测试三种 t2i 模型
支持从 .env 文件读取认证信息
"""

import os
import sys
import subprocess
import time
import requests
import json
import signal
from typing import Dict, Any, Optional, List
from pathlib import Path

# 添加项目根目录到 Python 路径
project_root = Path(__file__).parent.parent.parent
sys.path.insert(0, str(project_root))

# 加载环境变量
def load_env_file(env_path: Path) -> Dict[str, str]:
    """加载 .env 文件"""
    env_vars = {}
    if env_path.exists():
        with open(env_path, 'r', encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    env_vars[key.strip()] = value.strip()
    return env_vars

# 加载环境变量
env_file = project_root / ".env"
env_vars = load_env_file(env_file)

# 配置（优先使用环境变量，然后是 .env 文件，最后是默认值）
API_URL = os.getenv("API_URL", env_vars.get("API_URL", "http://localhost:10000/v1/images/generations"))
TIMEOUT = int(os.getenv("TIMEOUT", env_vars.get("TIMEOUT", "60")))
SERVICE_TIMEOUT = int(os.getenv("SERVICE_TIMEOUT", env_vars.get("SERVICE_TIMEOUT", "60")))

# 认证配置
AUTH_TOKEN = os.getenv("AUTH_TOKEN", env_vars.get("AUTH_TOKEN", ""))
API_KEY = os.getenv("API_KEY", env_vars.get("API_KEY", ""))
AUTH_USERNAME = os.getenv("AUTH_USERNAME", env_vars.get("AUTH_USERNAME", ""))
AUTH_PASSWORD = os.getenv("AUTH_PASSWORD", env_vars.get("AUTH_PASSWORD", ""))
CUSTOM_AUTH_HEADER = os.getenv("CUSTOM_AUTH_HEADER", env_vars.get("CUSTOM_AUTH_HEADER", ""))
CUSTOM_AUTH_VALUE = os.getenv("CUSTOM_AUTH_VALUE", env_vars.get("CUSTOM_AUTH_VALUE", ""))

MAKEFILE_PATH = project_root / "Makefile"
DEPLOY_DIR = project_root / "deploy_dev"
COMPOSE_FILE = DEPLOY_DIR / "docker-compose.yml"

# 模型配置
MODELS = {
    "doubao": {
        "model": "volcengine/volcengine/doubao-seedream-4.5",
        "model_type": "t2i",
        "name": "豆包 SeeDream 4.5",
        "make_target": "dev-t2i-doubao-d",
        "envoy_config": "envoy.yaml"
    },
    "qwen": {
        "model": "aliyun/aliyun/qwen-image-plus",
        "model_type": "t2i",
        "name": "通义千问图像生成",
        "make_target": "dev-t2i-qwen-d",
        "envoy_config": "envoy-qwen.yaml"
    },
    "gemini": {
        "model": "openrouter/auto/gemini-3-image",
        "model_type": "t2i",
        "name": "Gemini 3 Image",
        "make_target": "dev-t2i-gemini-d",
        "envoy_config": "envoy-gemini.yaml"
    }
}

# 测试用例
TEST_CASES = [
    {
        "name": "简单风景画",
        "prompt": "生成一幅美丽的风景画，包含山川、河流和蓝天白云",
        "prompt_en": "Generate a beautiful landscape painting with mountains, rivers, and blue sky with white clouds"
    },
    {
        "name": "现代建筑",
        "prompt": "设计一座现代化的摩天大楼，具有玻璃幕墙和独特的几何造型",
        "prompt_en": "Design a modern skyscraper with glass curtain walls and unique geometric shapes"
    }
]

class ModelTester:
    def __init__(self):
        self.current_process = None
        self.setup_signal_handlers()
        self.print_config()
    
    def print_config(self):
        """打印配置信息"""
        print("🔧 配置信息:")
        print(f"  API URL: {API_URL}")
        print(f"  请求超时: {TIMEOUT}s")
        print(f"  服务启动超时: {SERVICE_TIMEOUT}s")
        
        # 打印认证配置（隐藏敏感信息）
        auth_methods = []
        if AUTH_TOKEN:
            auth_methods.append(f"Bearer Token: {AUTH_TOKEN[:10]}...")
        if API_KEY:
            auth_methods.append(f"API Key: {API_KEY[:10]}...")
        if AUTH_USERNAME and AUTH_PASSWORD:
            auth_methods.append(f"Basic Auth: {AUTH_USERNAME}:***")
        if CUSTOM_AUTH_HEADER and CUSTOM_AUTH_VALUE:
            auth_methods.append(f"Custom Header: {CUSTOM_AUTH_HEADER}")
        
        if auth_methods:
            print(f"  认证方式: {', '.join(auth_methods)}")
        else:
            print("  认证方式: 无")
        
        print("")
    
    def setup_signal_handlers(self):
        """设置信号处理器"""
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
    
    def signal_handler(self, signum, frame):
        """信号处理器"""
        print(f"\n收到信号 {signum}，正在清理...")
        self.cleanup()
        sys.exit(0)
    
    def cleanup(self):
        """清理资源"""
        print("正在停止服务...")
        try:
            subprocess.run(
                ["make", "stop"],
                cwd=project_root,
                check=False,
                capture_output=True
            )
        except Exception as e:
            print(f"停止服务时出错: {e}")
    
    def get_auth_headers(self) -> Dict[str, str]:
        """获取认证头"""
        headers = {}
        
        # Bearer Token 认证
        if AUTH_TOKEN:
            headers["Authorization"] = f"Bearer {AUTH_TOKEN}"
        
        # API Key 认证
        if API_KEY:
            headers["X-API-Key"] = API_KEY
        
        # 自定义认证头
        if CUSTOM_AUTH_HEADER and CUSTOM_AUTH_VALUE:
            headers[CUSTOM_AUTH_HEADER] = CUSTOM_AUTH_VALUE
        
        return headers
    
    def get_auth_for_requests(self) -> Optional[tuple]:
        """获取 requests 库的认证信息"""
        if AUTH_USERNAME and AUTH_PASSWORD:
            return (AUTH_USERNAME, AUTH_PASSWORD)
        return None
    
    def run_make_command(self, target: str) -> bool:
        """运行 make 命令"""
        try:
            print(f"执行: make {target}")
            result = subprocess.run(
                ["make", target],
                cwd=project_root,
                check=True,
                capture_output=True,
                text=True
            )
            print(f"Make 命令执行成功")
            return True
        except subprocess.CalledProcessError as e:
            print(f"Make 命令执行失败: {e}")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            return False
    
    def wait_for_service(self, timeout: int = 60) -> bool:
        """等待服务启动"""
        print(f"等待服务启动 (超时: {timeout}s)...")
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            try:
                # 构建健康检查请求头
                headers = self.get_auth_headers()
                auth = self.get_auth_for_requests()
                
                # 先尝试主要的 API 端点，而不是健康检查端点
                response = requests.get(
                    "http://localhost:10000", 
                    timeout=5,
                    headers=headers,
                    auth=auth
                )
                # 只要能连接到服务就认为启动成功，不管返回什么状态码
                # 401, 403, 404 等都表示服务在运行
                print("✅ 服务已启动")
                return True
            except requests.exceptions.ConnectionError as e:
                # 连接被拒绝，服务还没启动
                if "Connection refused" in str(e):
                    pass  # 继续等待
                else:
                    # 其他连接错误，可能服务已启动
                    print("✅ 服务已启动")
                    return True
            except requests.exceptions.Timeout:
                # 超时可能表示服务正在启动
                pass
            except requests.exceptions.RequestException:
                # 其他请求异常，但服务可能已经启动
                print("✅ 服务已启动")
                return True
            
            print(".", end="", flush=True)
            time.sleep(2)
        
        print(f"\n❌ 服务启动超时 ({timeout}s)")
        
        # 超时后尝试检查 docker-compose 状态
        try:
            result = subprocess.run(
                ["docker-compose", "ps"],
                cwd=project_root / "deploy_dev",
                capture_output=True,
                text=True,
                timeout=10
            )
            print("Docker Compose 状态:")
            print(result.stdout)
            if result.stderr:
                print("错误信息:")
                print(result.stderr)
        except Exception as e:
            print(f"无法获取 docker-compose 状态: {e}")
        
        return False
    
    def create_request_payload(self, model_key: str, prompt: str) -> Dict[str, Any]:
        """创建请求载荷"""
        model_config = MODELS[model_key]
        
        # 根据模型选择合适的提示词
        text_prompt = prompt
        
        # 基础请求结构
        payload = {
            "model": model_config["model"],
            "input": {
                "messages": [
                    {
                        "role": "user",
                        "content": [
                            {
                                "text": text_prompt
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
        
        # 根据不同模型添加特定参数
        if model_key == "doubao":
            payload["extra_body"] = {
                "doubao_seedream": {
                    "sequential_image_generation": "auto",
                    "response_format": "url",
                    "stream": False
                }
            }
            payload["parameters"]["watermark"] = False
        elif model_key == "qwen":
            payload["parameters"]["watermark"] = False
            payload["parameters"]["prompt_extend"] = True
        elif model_key == "gemini":
            payload["extra_body"] = {
                "gemini_3_image": {
                    "modalities": ["image"],
                    "provider": {
                        "only": ["Google"]
                    }
                }
            }
            # Gemini 使用英文提示词
            payload["input"]["messages"][0]["content"][0]["text"] = TEST_CASES[0]["prompt_en"] if "风景画" in text_prompt else TEST_CASES[1]["prompt_en"]
        
        return payload
    
    def send_request(self, model_key: str, payload: Dict[str, Any], timeout: int = 30) -> Dict[str, Any]:
        """发送请求"""
        model_config = MODELS[model_key]
        
        # 基础请求头
        headers = {
            "Content-Type": "application/json",
            "model": model_config["model"],
            "model_type": model_config["model_type"]
        }
        
        # 添加认证头
        auth_headers = self.get_auth_headers()
        headers.update(auth_headers)
        
        # 获取认证信息
        auth = self.get_auth_for_requests()
        
        try:
            print(f"  发送请求到 {model_config['name']}...")
            if auth_headers:
                print(f"  使用认证头: {list(auth_headers.keys())}")
            if auth:
                print(f"  使用基础认证: {auth[0]}:***")
            
            start_time = time.time()
            
            response = requests.post(
                API_URL,
                json=payload,
                headers=headers,
                auth=auth,
                timeout=timeout
            )
            
            end_time = time.time()
            response_time = end_time - start_time
            
            return {
                "success": True,
                "status_code": response.status_code,
                "response_time": response_time,
                "data": response.json() if response.headers.get('content-type', '').startswith('application/json') else response.text,
                "headers": dict(response.headers)
            }
        
        except requests.exceptions.Timeout:
            return {
                "success": False,
                "error": "请求超时",
                "response_time": timeout
            }
        except requests.exceptions.RequestException as e:
            return {
                "success": False,
                "error": f"请求异常: {str(e)}",
                "response_time": 0
            }
        except Exception as e:
            return {
                "success": False,
                "error": f"未知错误: {str(e)}",
                "response_time": 0
            }
    
    def print_result(self, model_key: str, test_case: Dict[str, str], result: Dict[str, Any]):
        """打印测试结果"""
        model_name = MODELS[model_key]["name"]
        
        print(f"\n{'='*60}")
        print(f"模型: {model_name}")
        print(f"测试用例: {test_case['name']}")
        print(f"提示词: {test_case['prompt']}")
        print(f"{'='*60}")
        
        if result["success"]:
            print(f"✅ 请求成功")
            print(f"状态码: {result['status_code']}")
            print(f"响应时间: {result['response_time']:.2f}s")
            
            # 检查响应头
            if "model" in result["headers"]:
                print(f"响应模型: {result['headers']['model']}")
            if "model_type" in result["headers"]:
                print(f"响应模型类型: {result['headers']['model_type']}")
            if "x-transform-fail-stage" in result["headers"]:
                print(f"⚠️  转换失败阶段: {result['headers']['x-transform-fail-stage']}")
            
            # 分析响应数据
            data = result["data"]
            if isinstance(data, dict):
                if "request_id" in data:
                    print(f"请求ID: {data['request_id']}")
                
                if "output" in data and "choices" in data["output"]:
                    choices = data["output"]["choices"]
                    if choices and len(choices) > 0:
                        choice = choices[0]
                        print(f"完成原因: {choice.get('finish_reason', 'unknown')}")
                        
                        if "message" in choice and "content" in choice["message"]:
                            content = choice["message"]["content"]
                            image_count = len([c for c in content if "image" in c])
                            print(f"生成图片数量: {image_count}")
                            
                            # 显示图片信息
                            for i, c in enumerate(content):
                                if "image" in c:
                                    image_url = c["image"]
                                    if image_url.startswith("data:image"):
                                        print(f"图片 {i+1}: Base64 编码 (长度: {len(image_url)} 字符)")
                                    else:
                                        print(f"图片 {i+1}: {image_url[:100]}...")
                
                if "usage" in data:
                    usage = data["usage"]
                    if "image_count" in usage:
                        print(f"使用统计 - 图片数量: {usage['image_count']}")
                    if "height" in usage and "width" in usage:
                        print(f"使用统计 - 尺寸: {usage['width']}x{usage['height']}")
                
                # 检查错误格式
                if "code" in data and "message" in data:
                    print(f"❌ 错误响应:")
                    print(f"错误码: {data['code']}")
                    print(f"错误信息: {json.dumps(data['message'], ensure_ascii=False, indent=2)}")
            else:
                print(f"响应数据: {data}")
        
        else:
            print(f"❌ 请求失败")
            print(f"错误: {result['error']}")
            print(f"响应时间: {result['response_time']:.2f}s")
    
    def test_model(self, model_key: str) -> bool:
        """测试单个模型"""
        if model_key not in MODELS:
            print(f"❌ 不支持的模型: {model_key}")
            return False
        
        model_config = MODELS[model_key]
        
        print(f"\n🚀 开始测试模型: {model_config['name']}")
        print(f"Make 目标: {model_config['make_target']}")
        print(f"Envoy 配置: {model_config['envoy_config']}")
        
        # 1. 停止当前服务
        print("\n1. 停止当前服务...")
        self.run_make_command("stop")
        time.sleep(3)
        
        # 2. 启动新的服务配置
        print(f"\n2. 启动 {model_config['name']} 服务...")
        if not self.run_make_command(model_config['make_target']):
            print(f"❌ 启动 {model_config['name']} 服务失败")
            return False
        
        # 3. 等待服务启动
        if not self.wait_for_service(SERVICE_TIMEOUT):
            print(f"❌ {model_config['name']} 服务启动超时")
            return False
        
        # 4. 执行测试用例
        print(f"\n3. 执行测试用例...")
        success_count = 0
        total_count = len(TEST_CASES)
        
        for i, test_case in enumerate(TEST_CASES):
            print(f"\n测试用例 {i+1}/{total_count}: {test_case['name']}")
            
            payload = self.create_request_payload(model_key, test_case["prompt"])
            result = self.send_request(model_key, payload, TIMEOUT)
            self.print_result(model_key, test_case, result)
            
            if result["success"] and result["status_code"] == 200:
                success_count += 1
            
            # 测试间隔
            if i < total_count - 1:
                time.sleep(3)
        
        print(f"\n📊 {model_config['name']} 测试结果: {success_count}/{total_count} 成功")
        return success_count == total_count
    
    def test_all_models(self) -> Dict[str, bool]:
        """测试所有模型"""
        print("🚀 开始测试所有 T2I 模型...")
        print(f"项目根目录: {project_root}")
        print(f"API 端点: {API_URL}")
        
        results = {}
        
        for model_key in MODELS.keys():
            print(f"\n{'='*80}")
            success = self.test_model(model_key)
            results[model_key] = success
            
            if not success:
                print(f"⚠️  {MODELS[model_key]['name']} 测试失败，继续测试下一个模型...")
            
            # 模型间等待时间
            time.sleep(5)
        
        return results
    
    def print_summary(self, results: Dict[str, bool]):
        """打印测试总结"""
        print(f"\n{'='*80}")
        print("📊 测试总结")
        print(f"{'='*80}")
        
        success_count = 0
        total_count = len(results)
        
        for model_key, success in results.items():
            model_name = MODELS[model_key]["name"]
            status = "✅ 成功" if success else "❌ 失败"
            print(f"{model_name}: {status}")
            if success:
                success_count += 1
        
        print(f"\n总体结果: {success_count}/{total_count} 模型测试成功")
        
        if success_count == total_count:
            print("🎉 所有模型测试通过!")
        else:
            print("⚠️  部分模型测试失败，请检查日志")

def main():
    """主函数"""
    # 检查 .env 文件
    env_file = project_root / ".env"
    if not env_file.exists():
        print("⚠️  未找到 .env 文件")
        print(f"请复制 .env.example 为 .env 并配置认证信息:")
        print(f"  cp {project_root}/.env.example {project_root}/.env")
        print("然后编辑 .env 文件填入认证信息")
        print("")
        
        # 询问是否继续（无认证）
        try:
            response = input("是否继续测试（无认证）？[y/N]: ").strip().lower()
            if response not in ['y', 'yes']:
                print("测试已取消")
                sys.exit(0)
        except KeyboardInterrupt:
            print("\n测试已取消")
            sys.exit(0)
    
    tester = ModelTester()
    
    try:
        # 检查项目结构
        if not MAKEFILE_PATH.exists():
            print(f"❌ Makefile 不存在: {MAKEFILE_PATH}")
            sys.exit(1)
        
        if not DEPLOY_DIR.exists():
            print(f"❌ 部署目录不存在: {DEPLOY_DIR}")
            sys.exit(1)
        
        # 执行测试
        results = tester.test_all_models()
        
        # 打印总结
        tester.print_summary(results)
        
        # 清理
        tester.cleanup()
        
        # 根据结果设置退出码
        success_count = sum(results.values())
        if success_count == len(results):
            print("\n✅ 所有测试完成!")
            sys.exit(0)
        else:
            print(f"\n❌ 测试失败: {len(results) - success_count} 个模型测试失败")
            sys.exit(1)
    
    except KeyboardInterrupt:
        print("\n用户中断测试")
        tester.cleanup()
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ 测试过程中出现异常: {e}")
        tester.cleanup()
        sys.exit(1)

if __name__ == "__main__":
    main()

# 模型配置
MODELS = {
    "doubao": {
        "model": "volcengine/volcengine/doubao-seedream-4.5",
        "model_type": "t2i",
        "name": "豆包 SeeDream 4.5",
        "make_target": "dev-t2i-doubao-d",
        "envoy_config": "envoy.yaml"
    },
    "qwen": {
        "model": "aliyun/aliyun/qwen-image-plus",
        "model_type": "t2i",
        "name": "通义千问图像生成",
        "make_target": "dev-t2i-qwen-d",
        "envoy_config": "envoy-qwen.yaml"
    },
    "gemini": {
        "model": "openrouter/auto/gemini-3-image",
        "model_type": "t2i",
        "name": "Gemini 3 Image",
        "make_target": "dev-t2i-gemini-d",
        "envoy_config": "envoy-gemini.yaml"
    }
}

# 测试用例
TEST_CASES = [
    {
        "name": "简单风景画",
        "prompt": "生成一幅美丽的风景画，包含山川、河流和蓝天白云",
        "prompt_en": "Generate a beautiful landscape painting with mountains, rivers, and blue sky with white clouds"
    },
    {
        "name": "现代建筑",
        "prompt": "设计一座现代化的摩天大楼，具有玻璃幕墙和独特的几何造型",
        "prompt_en": "Design a modern skyscraper with glass curtain walls and unique geometric shapes"
    }
]

class ModelTester:
    def __init__(self):
        self.current_process = None
        self.setup_signal_handlers()
    
    def setup_signal_handlers(self):
        """设置信号处理器"""
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
    
    def signal_handler(self, signum, frame):
        """信号处理器"""
        print(f"\n收到信号 {signum}，正在清理...")
        self.cleanup()
        sys.exit(0)
    
    def cleanup(self):
        """清理资源"""
        print("正在停止服务...")
        try:
            subprocess.run(
                ["make", "stop"],
                cwd=project_root,
                check=False,
                capture_output=True
            )
        except Exception as e:
            print(f"停止服务时出错: {e}")
    
    def run_make_command(self, target: str) -> bool:
        """运行 make 命令"""
        try:
            print(f"执行: make {target}")
            result = subprocess.run(
                ["make", target],
                cwd=project_root,
                check=True,
                capture_output=True,
                text=True
            )
            print(f"Make 命令执行成功")
            return True
        except subprocess.CalledProcessError as e:
            print(f"Make 命令执行失败: {e}")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            return False
    
    def wait_for_service(self, timeout: int = 60) -> bool:
        """等待服务启动"""
        print(f"等待服务启动 (超时: {timeout}s)...")
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            try:
                response = requests.get("http://localhost:10000/health", timeout=5)
                if response.status_code == 200:
                    print("✅ 服务已启动")
                    return True
            except requests.exceptions.RequestException:
                pass
            
            print(".", end="", flush=True)
            time.sleep(2)
        
        print(f"\n❌ 服务启动超时 ({timeout}s)")
        return False
    
    def create_request_payload(self, model_key: str, prompt: str) -> Dict[str, Any]:
        """创建请求载荷"""
        model_config = MODELS[model_key]
        
        # 根据模型选择合适的提示词
        text_prompt = prompt
        
        # 基础请求结构
        payload = {
            "model": model_config["model"],
            "input": {
                "messages": [
                    {
                        "role": "user",
                        "content": [
                            {
                                "text": text_prompt
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
        
        # 根据不同模型添加特定参数
        if model_key == "doubao":
            payload["extra_body"] = {
                "doubao_seedream": {
                    "sequential_image_generation": "auto",
                    "response_format": "url",
                    "stream": False
                }
            }
            payload["parameters"]["watermark"] = False
        elif model_key == "qwen":
            payload["parameters"]["watermark"] = False
            payload["parameters"]["prompt_extend"] = True
        elif model_key == "gemini":
            payload["extra_body"] = {
                "gemini_3_image": {
                    "modalities": ["image"],
                    "provider": {
                        "only": ["Google"]
                    }
                }
            }
            # Gemini 使用英文提示词
            payload["input"]["messages"][0]["content"][0]["text"] = TEST_CASES[0]["prompt_en"] if "风景画" in text_prompt else TEST_CASES[1]["prompt_en"]
        
        return payload
    
    def send_request(self, model_key: str, payload: Dict[str, Any], timeout: int = 30) -> Dict[str, Any]:
        """发送请求"""
        model_config = MODELS[model_key]
        
        headers = {
            "Content-Type": "application/json",
            "model": model_config["model"],
            "model_type": model_config["model_type"]
        }
        
        try:
            print(f"  发送请求到 {model_config['name']}...")
            start_time = time.time()
            
            response = requests.post(
                API_URL,
                json=payload,
                headers=headers,
                timeout=timeout
            )
            
            end_time = time.time()
            response_time = end_time - start_time
            
            return {
                "success": True,
                "status_code": response.status_code,
                "response_time": response_time,
                "data": response.json() if response.headers.get('content-type', '').startswith('application/json') else response.text,
                "headers": dict(response.headers)
            }
        
        except requests.exceptions.Timeout:
            return {
                "success": False,
                "error": "请求超时",
                "response_time": timeout
            }
        except requests.exceptions.RequestException as e:
            return {
                "success": False,
                "error": f"请求异常: {str(e)}",
                "response_time": 0
            }
        except Exception as e:
            return {
                "success": False,
                "error": f"未知错误: {str(e)}",
                "response_time": 0
            }
    
    def print_result(self, model_key: str, test_case: Dict[str, str], result: Dict[str, Any]):
        """打印测试结果"""
        model_name = MODELS[model_key]["name"]
        
        print(f"\n{'='*60}")
        print(f"模型: {model_name}")
        print(f"测试用例: {test_case['name']}")
        print(f"提示词: {test_case['prompt']}")
        print(f"{'='*60}")
        
        if result["success"]:
            print(f"✅ 请求成功")
            print(f"状态码: {result['status_code']}")
            print(f"响应时间: {result['response_time']:.2f}s")
            
            # 检查响应头
            if "model" in result["headers"]:
                print(f"响应模型: {result['headers']['model']}")
            if "model_type" in result["headers"]:
                print(f"响应模型类型: {result['headers']['model_type']}")
            if "x-transform-fail-stage" in result["headers"]:
                print(f"⚠️  转换失败阶段: {result['headers']['x-transform-fail-stage']}")
            
            # 分析响应数据
            data = result["data"]
            if isinstance(data, dict):
                if "request_id" in data:
                    print(f"请求ID: {data['request_id']}")
                
                if "output" in data and "choices" in data["output"]:
                    choices = data["output"]["choices"]
                    if choices and len(choices) > 0:
                        choice = choices[0]
                        print(f"完成原因: {choice.get('finish_reason', 'unknown')}")
                        
                        if "message" in choice and "content" in choice["message"]:
                            content = choice["message"]["content"]
                            image_count = len([c for c in content if "image" in c])
                            print(f"生成图片数量: {image_count}")
                            
                            # 显示图片信息
                            for i, c in enumerate(content):
                                if "image" in c:
                                    image_url = c["image"]
                                    if image_url.startswith("data:image"):
                                        print(f"图片 {i+1}: Base64 编码 (长度: {len(image_url)} 字符)")
                                    else:
                                        print(f"图片 {i+1}: {image_url[:100]}...")
                
                if "usage" in data:
                    usage = data["usage"]
                    if "image_count" in usage:
                        print(f"使用统计 - 图片数量: {usage['image_count']}")
                    if "height" in usage and "width" in usage:
                        print(f"使用统计 - 尺寸: {usage['width']}x{usage['height']}")
                
                # 检查错误格式
                if "code" in data and "message" in data:
                    print(f"❌ 错误响应:")
                    print(f"错误码: {data['code']}")
                    print(f"错误信息: {json.dumps(data['message'], ensure_ascii=False, indent=2)}")
            else:
                print(f"响应数据: {data}")
        
        else:
            print(f"❌ 请求失败")
            print(f"错误: {result['error']}")
            print(f"响应时间: {result['response_time']:.2f}s")
    
    def test_model(self, model_key: str) -> bool:
        """测试单个模型"""
        if model_key not in MODELS:
            print(f"❌ 不支持的模型: {model_key}")
            return False
        
        model_config = MODELS[model_key]
        
        print(f"\n🚀 开始测试模型: {model_config['name']}")
        print(f"Make 目标: {model_config['make_target']}")
        print(f"Envoy 配置: {model_config['envoy_config']}")
        
        # 1. 停止当前服务
        print("\n1. 停止当前服务...")
        self.run_make_command("stop")
        time.sleep(3)
        
        # 2. 启动新的服务配置
        print(f"\n2. 启动 {model_config['name']} 服务...")
        if not self.run_make_command(model_config['make_target']):
            print(f"❌ 启动 {model_config['name']} 服务失败")
            return False
        
        # 3. 等待服务启动
        if not self.wait_for_service(60):
            print(f"❌ {model_config['name']} 服务启动超时")
            return False
        
        # 4. 执行测试用例
        print(f"\n3. 执行测试用例...")
        success_count = 0
        total_count = len(TEST_CASES)
        
        for i, test_case in enumerate(TEST_CASES):
            print(f"\n测试用例 {i+1}/{total_count}: {test_case['name']}")
            
            payload = self.create_request_payload(model_key, test_case["prompt"])
            result = self.send_request(model_key, payload, 30)
            self.print_result(model_key, test_case, result)
            
            if result["success"] and result["status_code"] == 200:
                success_count += 1
            
            # 测试间隔
            if i < total_count - 1:
                time.sleep(3)
        
        print(f"\n📊 {model_config['name']} 测试结果: {success_count}/{total_count} 成功")
        return success_count == total_count
    
    def test_all_models(self) -> Dict[str, bool]:
        """测试所有模型"""
        print("🚀 开始测试所有 T2I 模型...")
        print(f"项目根目录: {project_root}")
        print(f"API 端点: {API_URL}")
        
        results = {}
        
        for model_key in MODELS.keys():
            print(f"\n{'='*80}")
            success = self.test_model(model_key)
            results[model_key] = success
            
            if not success:
                print(f"⚠️  {MODELS[model_key]['name']} 测试失败，继续测试下一个模型...")
            
            # 模型间等待时间
            time.sleep(5)
        
        return results
    
    def print_summary(self, results: Dict[str, bool]):
        """打印测试总结"""
        print(f"\n{'='*80}")
        print("📊 测试总结")
        print(f"{'='*80}")
        
        success_count = 0
        total_count = len(results)
        
        for model_key, success in results.items():
            model_name = MODELS[model_key]["name"]
            status = "✅ 成功" if success else "❌ 失败"
            print(f"{model_name}: {status}")
            if success:
                success_count += 1
        
        print(f"\n总体结果: {success_count}/{total_count} 模型测试成功")
        
        if success_count == total_count:
            print("🎉 所有模型测试通过!")
        else:
            print("⚠️  部分模型测试失败，请检查日志")

def main():
    """主函数"""
    tester = ModelTester()
    
    try:
        # 检查项目结构
        if not MAKEFILE_PATH.exists():
            print(f"❌ Makefile 不存在: {MAKEFILE_PATH}")
            sys.exit(1)
        
        if not DEPLOY_DIR.exists():
            print(f"❌ 部署目录不存在: {DEPLOY_DIR}")
            sys.exit(1)
        
        # 执行测试
        results = tester.test_all_models()
        
        # 打印总结
        tester.print_summary(results)
        
        # 清理
        tester.cleanup()
        
        # 根据结果设置退出码
        success_count = sum(results.values())
        if success_count == len(results):
            print("\n✅ 所有测试完成!")
            sys.exit(0)
        else:
            print(f"\n❌ 测试失败: {len(results) - success_count} 个模型测试失败")
            sys.exit(1)
    
    except KeyboardInterrupt:
        print("\n用户中断测试")
        tester.cleanup()
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ 测试过程中出现异常: {e}")
        tester.cleanup()
        sys.exit(1)

if __name__ == "__main__":
    main()