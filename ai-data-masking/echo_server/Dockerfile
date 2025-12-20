FROM python:3.11-slim

WORKDIR /app

# 安装依赖
RUN pip install --no-cache-dir fastapi uvicorn

# 复制应用文件
COPY echo.py .

# 暴露端口
EXPOSE 8000

# 启动应用
CMD ["uvicorn", "echo:app", "--host", "0.0.0.0", "--port", "8000"]

