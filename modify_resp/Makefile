.PHONY: build deploy all

# build: 执行 dev.sh 的第 2-3 行（go mod tidy 和 go build）
build:
	go mod tidy
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

# deploy: 执行 dev.sh 的第 6-7 行（docker-compose down 和 up）
deploy:
	docker-compose -f deploy_dev/docker-compose.yml down
	docker-compose -f deploy_dev/docker-compose.yml up -d

# all:
all: build
	cp main.wasm deploy_dev/main.wasm
	$(MAKE) deploy

# all: 
all_dev: build
	rm -f deploy_dev/main.wasm
	cp main.wasm deploy_dev/main.wasm
	docker-compose -f deploy_dev/docker-compose.yml down
	docker-compose -f deploy_dev/docker-compose.yml up