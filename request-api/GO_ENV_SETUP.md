# Go 环境设置指南

## 设置依赖包下载到 pkg 目录

### 方法 1：设置 GOMODCACHE 环境变量（推荐）

```bash
# 设置模块缓存目录到 pkg/mod
export GOMODCACHE=/data/work/go/pkg/mod

# 或者设置为项目相对路径
export GOMODCACHE=$(pwd)/pkg/mod

# 永久设置（添加到 ~/.bashrc 或 ~/.zshrc）
echo 'export GOMODCACHE=/data/work/go/pkg/mod' >> ~/.bashrc
source ~/.bashrc
```

### 方法 2：设置 GOPATH 并启用 Go Modules

```bash
# 设置 GOPATH
export GOPATH=/data/work/go

# 启用 Go Modules（即使项目在 GOPATH 下）
export GO111MODULE=on

# 设置模块缓存目录
export GOMODCACHE=$GOPATH/pkg/mod

# 永久设置
cat >> ~/.bashrc << 'EOF'
export GOPATH=/data/work/go
export GO111MODULE=on
export GOMODCACHE=$GOPATH/pkg/mod
EOF
source ~/.bashrc
```

### 方法 3：使用 go env 命令设置

```bash
# 设置 GOMODCACHE
go env -w GOMODCACHE=/data/work/go/pkg/mod

# 设置 GOPROXY（可选，使用国内镜像加速）
go env -w GOPROXY=https://goproxy.cn,direct

# 查看当前设置
go env GOMODCACHE
```

### 验证设置

```bash
# 查看 Go 环境变量
go env | grep -E "GOPATH|GOMODCACHE|GO111MODULE"

# 下载依赖到指定目录
cd /data/work/go/higress-wasm-demo-go/http-call
unset GOPATH  # 临时取消 GOPATH，避免警告
go mod download

# 检查依赖是否下载到 pkg/mod 目录
ls -la /data/work/go/pkg/mod/github.com/higress-group/
```

### 当前项目设置

根据当前环境，推荐使用以下设置：

```bash
# 在项目目录下执行
cd /data/work/go/higress-wasm-demo-go/http-call

# 方法 1：临时设置（当前终端会话有效）
export GO111MODULE=on
export GOMODCACHE=/data/work/go/pkg/mod
unset GOPATH  # 或者将项目移出 GOPATH 目录

# 方法 2：使用 go env 永久设置
go env -w GOMODCACHE=/data/work/go/pkg/mod
go env -w GO111MODULE=on
```

### 下载依赖

```bash
# 进入项目目录
cd /data/work/go/higress-wasm-demo-go/http-call

# 临时取消 GOPATH（避免警告）
unset GOPATH

# 下载所有依赖
go mod download

# 或者使用 tidy 自动整理依赖
go mod tidy

# 验证依赖位置
ls -la /data/work/go/pkg/mod/github.com/higress-group/
```

### 常见问题

1. **警告：ignoring go.mod in $GOPATH**
   - 解决：设置 `export GO111MODULE=on` 或 `unset GOPATH`

2. **依赖包下载位置**
   - Go Modules 模式下，依赖默认下载到 `$GOMODCACHE`（通常是 `$GOPATH/pkg/mod`）
   - 可以通过 `go env GOMODCACHE` 查看当前设置

3. **使用国内镜像加速**
   ```bash
   go env -w GOPROXY=https://goproxy.cn,direct
   ```

