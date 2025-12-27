.PHONY: build deploy all docker-build docker-login docker-push

# 镜像名称和标签
IMAGE_NAME ?= model.vnet.com/vnet-tech-public/higress-wasm-demo-go
IMAGE_TAG ?= latest

# build: 执行 dev.sh 的第 2-3 行（go mod tidy 和 go build）
build:
	go mod tidy
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

# docker-build: 构建 Docker 镜像
docker-build: build
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) -f Dockerfile .

# docker-login: 登录镜像仓库
docker-login:
	@echo "7FfNlITt1PcEg6cvKWxj8CdeDviEmsPX" | docker login model.vnet.com -u robot\$$vnet-tech-public+robot --password-stdin

deploy: build
	ssh -p 50022 172.22.221.214 'mkdir -p /data/work/deploy/AgenSpace/plugin-server/plugins/hello-world/1.0.0 && chmod 755 /data/work/deploy/AgenSpace/plugin-server/plugins/hello-world/1.0.0'
	scp -P 50022 main.wasm 172.22.221.214:/data/work/deploy/AgenSpace/plugin-server/plugins/hello-world/1.0.0/plugin.wasm
	ssh -p 50022 172.22.221.214 'pushd /data/work/deploy/AgenSpace/plugin-server && python3 generate_metadata.py'

# all: 执行 dev.sh 中的所有逻辑
all: build
	cp main.wasm deploy_dev/main.wasm
	$(MAKE) deploy

# all: 执行 dev.sh 中的所有逻辑
all_dev: build
	cp main.wasm deploy_dev/main.wasm
	docker-compose -f deploy_dev/docker-compose.yml down
	docker-compose -f deploy_dev/docker-compose.yml up
