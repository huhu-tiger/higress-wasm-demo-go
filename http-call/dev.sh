#!/bin/bash
go mod tidy
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./

cp main.wasm deploy_dev/main.wasm
docker-compose -f deploy_dev/docker-compose.yml down
docker-compose -f deploy_dev/docker-compose.yml up -d