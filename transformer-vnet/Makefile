.PHONY: build deploy all dev-t2i-qwen dev-t2i-doubao dev-t2i-gemini dev-t2i-qwen-d dev-t2i-doubao-d dev-t2i-gemini-d switch-t2i-qwen switch-t2i-doubao switch-t2i-gemini dev-qwen dev-doubao dev-gemini dev-qwen-d dev-doubao-d dev-gemini-d switch-qwen switch-doubao switch-gemini switch-config clean help

# 配置文件路径
DEPLOY_DIR = deploy_dev
COMPOSE_FILE = $(DEPLOY_DIR)/docker-compose.yml
ENVOY_QWEN = $(DEPLOY_DIR)/t2i/envoy-qwen.yaml
ENVOY_DOUBAO = $(DEPLOY_DIR)/t2i/envoy.yaml
ENVOY_GEMINI = $(DEPLOY_DIR)/t2i/envoy-gemini.yaml

# build: 执行 dev.sh 的第 2-3 行（go mod tidy 和 go build）
build:
	go mod tidy
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

# deploy: 执行 dev.sh 的第 6-7 行（docker-compose down 和 up）
# deploy:
# 	docker-compose -f deploy_dev/docker-compose.yml down
# 	docker-compose -f deploy_dev/docker-compose.yml up -d
clean_build:
	rm -f main.wasm


deploy: build
	ssh -p 50022 172.22.221.214 'mkdir -p /data/work/deploy/AgenSpace/plugin-server/plugins/transformer-vnet/1.0.0 && chmod 755 /data/work/deploy/AgenSpace/plugin-server/plugins/transformer-vnet/1.0.0'
	scp -P 50022 main.wasm 172.22.221.214:/data/work/deploy/AgenSpace/plugin-server/plugins/transformer-vnet/1.0.0/plugin.wasm
	ssh -p 50022 172.22.221.214 'pushd /data/work/deploy/AgenSpace/plugin-server && python3 generate_metadata.py'
	

# all: 执行 dev.sh 中的所有逻辑
all: build
	cp main.wasm $(DEPLOY_DIR)/main.wasm
	$(MAKE) clean_build
	$(MAKE) deploy

# 切换 Envoy 配置文件的函数
# 用法: $(call switch-config,config-file)
define switch-config
	@echo "切换到配置: $(1)"
	@sed -i.bak "s|- .*envoy.*\.yaml:/etc/envoy/envoy.yaml|- ./t2i/$(1):/etc/envoy/envoy.yaml|" $(COMPOSE_FILE)
	@sed -i.bak "s|^[[:space:]]*\./.*envoy.*\.yaml:/etc/envoy/envoy.yaml|    - ./t2i/$(1):/etc/envoy/envoy.yaml|" $(COMPOSE_FILE)
	@rm -f $(COMPOSE_FILE).bak
endef

# 准备开发环境（构建并复制 wasm 文件）
prepare-dev: build
	@echo "准备开发环境..."
	@cp main.wasm $(DEPLOY_DIR)/main.wasm
	$(MAKE) clean_build

# ========== t2i 文生图相关命令 ==========

# dev-t2i-qwen: 使用 qwen-image-plus 配置启动开发环境（前台）
dev-t2i-qwen: prepare-dev
	@echo "启动 t2i/qwen-image-plus 配置..."
	$(call switch-config,envoy-qwen.yaml)
	@cd $(DEPLOY_DIR) && docker-compose down
	@cd $(DEPLOY_DIR) && docker-compose up

# dev-t2i-doubao: 使用 doubao-seedream-4.5 配置启动开发环境（前台，默认）
dev-t2i-doubao: prepare-dev
	@echo "启动 t2i/doubao-seedream-4.5 配置..."
	$(call switch-config,envoy.yaml)
	@cd $(DEPLOY_DIR) && docker-compose down
	@cd $(DEPLOY_DIR) && docker-compose up

# dev-t2i-gemini: 使用 gemini-3-image 配置启动开发环境（前台）
dev-t2i-gemini: prepare-dev
	@echo "启动 t2i/gemini-3-image 配置..."
	$(call switch-config,envoy-gemini.yaml)
	@cd $(DEPLOY_DIR) && docker-compose down
	@cd $(DEPLOY_DIR) && docker-compose up

# 后台启动命令（不阻塞）
dev-t2i-qwen-d: prepare-dev
	@echo "后台启动 t2i/qwen-image-plus 配置..."
	$(call switch-config,envoy-qwen.yaml)
	@cd $(DEPLOY_DIR) && docker-compose down
	@cd $(DEPLOY_DIR) && docker-compose up -d
	@echo "服务已启动，访问 http://localhost:10000"

dev-t2i-doubao-d: prepare-dev
	@echo "后台启动 t2i/doubao-seedream-4.5 配置..."
	$(call switch-config,envoy.yaml)
	@cd $(DEPLOY_DIR) && docker-compose down
	@cd $(DEPLOY_DIR) && docker-compose up -d
	@echo "服务已启动，访问 http://localhost:10000"

dev-t2i-gemini-d: prepare-dev
	@echo "后台启动 t2i/gemini-3-image 配置..."
	$(call switch-config,envoy-gemini.yaml)
	@cd $(DEPLOY_DIR) && docker-compose down
	@cd $(DEPLOY_DIR) && docker-compose up -d
	@echo "服务已启动，访问 http://localhost:10000"

# 仅切换配置（不启动服务）
switch-t2i-qwen:
	$(call switch-config,envoy-qwen.yaml)
	@echo "配置已切换为 t2i/qwen-image-plus，请手动重启服务"

switch-t2i-doubao:
	$(call switch-config,envoy.yaml)
	@echo "配置已切换为 t2i/doubao-seedream-4.5，请手动重启服务"

switch-t2i-gemini:
	$(call switch-config,envoy-gemini.yaml)
	@echo "配置已切换为 t2i/gemini-3-image，请手动重启服务"

# ========== 向后兼容的旧命令（别名） ==========

# dev-qwen: 使用 qwen-image-plus 配置启动开发环境（向后兼容）
dev-qwen: dev-t2i-qwen

# dev-doubao: 使用 doubao-seedream-4.5 配置启动开发环境（向后兼容）
dev-doubao: dev-t2i-doubao

# dev-gemini: 使用 gemini-3-image 配置启动开发环境（向后兼容）
dev-gemini: dev-t2i-gemini

# 后台启动命令（向后兼容）
dev-qwen-d: dev-t2i-qwen-d
dev-doubao-d: dev-t2i-doubao-d
dev-gemini-d: dev-t2i-gemini-d

# 仅切换配置（向后兼容）
switch-qwen: switch-t2i-qwen
switch-doubao: switch-t2i-doubao
switch-gemini: switch-t2i-gemini

# 停止服务
stop:
	@cd $(DEPLOY_DIR) && docker-compose down

# 清理
clean: stop
	@rm -f main.wasm
	@rm -f $(DEPLOY_DIR)/main.wasm
	@rm -f $(COMPOSE_FILE).bak

# 帮助信息
help:
	@echo "Transformer 插件 Makefile 命令："
	@echo ""
	@echo "构建命令："
	@echo "  make build              - 构建 WASM 文件"
	@echo "  make deploy             - 部署到远程服务器"
	@echo ""
	@echo "========== t2i 文生图相关命令 =========="
	@echo ""
	@echo "开发环境（前台运行）："
	@echo "  make dev-t2i-qwen       - 使用 t2i/qwen-image-plus 配置启动（前台）"
	@echo "  make dev-t2i-doubao     - 使用 t2i/doubao-seedream-4.5 配置启动（前台，默认）"
	@echo "  make dev-t2i-gemini     - 使用 t2i/gemini-3-image 配置启动（前台）"
	@echo ""
	@echo "开发环境（后台运行）："
	@echo "  make dev-t2i-qwen-d     - 使用 t2i/qwen-image-plus 配置启动（后台）"
	@echo "  make dev-t2i-doubao-d   - 使用 t2i/doubao-seedream-4.5 配置启动（后台）"
	@echo "  make dev-t2i-gemini-d   - 使用 t2i/gemini-3-image 配置启动（后台）"
	@echo ""
	@echo "配置切换（不启动服务）："
	@echo "  make switch-t2i-qwen    - 切换到 t2i/qwen-image-plus 配置"
	@echo "  make switch-t2i-doubao  - 切换到 t2i/doubao-seedream-4.5 配置"
	@echo "  make switch-t2i-gemini  - 切换到 t2i/gemini-3-image 配置"
	@echo ""
	@echo "========== 向后兼容的旧命令（别名） =========="
	@echo ""
	@echo "开发环境（前台运行）："
	@echo "  make dev-qwen           - 使用 qwen-image-plus 配置启动（前台，向后兼容）"
	@echo "  make dev-doubao         - 使用 doubao-seedream-4.5 配置启动（前台，向后兼容）"
	@echo "  make dev-gemini         - 使用 gemini-3-image 配置启动（前台，向后兼容）"
	@echo ""
	@echo "开发环境（后台运行）："
	@echo "  make dev-qwen-d         - 使用 qwen-image-plus 配置启动（后台，向后兼容）"
	@echo "  make dev-doubao-d       - 使用 doubao-seedream-4.5 配置启动（后台，向后兼容）"
	@echo "  make dev-gemini-d       - 使用 gemini-3-image 配置启动（后台，向后兼容）"
	@echo ""
	@echo "配置切换（不启动服务）："
	@echo "  make switch-qwen        - 切换到 qwen-image-plus 配置（向后兼容）"
	@echo "  make switch-doubao      - 切换到 doubao-seedream-4.5 配置（向后兼容）"
	@echo "  make switch-gemini      - 切换到 gemini-3-image 配置（向后兼容）"
	@echo ""
	@echo "其他命令："
	@echo "  make stop               - 停止服务"
	@echo "  make clean              - 清理构建文件和停止服务"
	@echo "  make help               - 显示此帮助信息"
	@echo ""
	@echo "示例："
	@echo "  make dev-t2i-doubao     - 启动 t2i/doubao-seedream-4.5 测试环境"
	@echo "  make dev-t2i-gemini-d   - 后台启动 t2i/gemini-3-image 测试环境"
	@echo "  make switch-t2i-qwen && make stop && cd deploy_dev && docker-compose up -d  - 切换配置并重启"
