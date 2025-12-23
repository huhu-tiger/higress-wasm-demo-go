.PHONY: build deploy all

# build: 执行 dev.sh 的第 2-3 行（go mod tidy 和 go build）
build:
	go mod tidy
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

# deploy: 执行 dev.sh 的第 6-7 行（docker-compose down 和 up）
# deploy:
# 	docker-compose -f deploy_dev/docker-compose.yml down
# 	docker-compose -f deploy_dev/docker-compose.yml up -d

deploy: build
	ssh -p 50022 172.22.221.214 'mkdir -p /data/work/deploy/AgenSpace/plugin-server/plugins/ai-data-masking-go/1.0.0 && chmod 755 /data/work/deploy/AgenSpace/plugin-server/plugins/ai-data-masking-go/1.0.0'
	scp -P 50022 main.wasm 172.22.221.214:/data/work/deploy/AgenSpace/plugin-server/plugins/ai-data-masking-go/1.0.0/plugin.wasm
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
