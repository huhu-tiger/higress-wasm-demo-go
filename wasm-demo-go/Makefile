.PHONY: build deploy all docker-build docker-login docker-push

# 镜像名称和标签
IMAGE_NAME ?= model.vnet.com/vnet-tech-public/higress-wasm-demo-go
IMAGE_TAG ?= latest

# build: 执行 dev.sh 的第 2-3 行（go mod tidy 和 go build）
build:
	go mod tidy
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

# deploy: 执行 dev.sh 的第 6-7 行（docker-compose down 和 up）
deploy:
	docker-compose -f deploy_dev/docker-compose.yml down
	docker-compose -f deploy_dev/docker-compose.yml up -d

# docker-build: 构建 Docker 镜像
docker-build: build
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) -f Dockerfile .

# docker-login: 登录镜像仓库
docker-login:
	@echo "7FfNlITt1PcEg6cvKWxj8CdeDviEmsPX" | docker login model.vnet.com -u robot\$$vnet-tech-public+robot --password-stdin

# docker-push: 推送 Docker 镜像到仓库
docker-push: docker-build docker-login
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

# all: 执行 dev.sh 中的所有逻辑
all: build
	cp main.wasm deploy_dev/main.wasm
	$(MAKE) deploy

