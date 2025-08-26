# 项目名称
PROJECT_NAME := yt-dlp-simpgo
# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go参数
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# 构建参数
LDFLAGS := -X 'main.Version=$(VERSION)' -s -w

.PHONY: build clean release test help

help: ## 显示帮助信息
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 构建项目
	go build -ldflags="$(LDFLAGS)" -o $(PROJECT_NAME) .

build-gui: ## 构建Windows GUI版本
	GOOS=windows go build -ldflags="$(LDFLAGS) -H=windowsgui" -o $(PROJECT_NAME)-windows-gui.exe .

clean: ## 清理构建产物
	rm -f $(PROJECT_NAME)
	rm -f $(PROJECT_NAME)-windows-gui.exe
	rm -f $(PROJECT_NAME)-*.tar.gz
	rm -f $(PROJECT_NAME)-*.zip

release: clean ## 构建发布版本
	# Windows GUI版本
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS) -H=windowsgui" -o $(PROJECT_NAME)-windows-amd64.exe .
	# Linux版本
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(PROJECT_NAME)-linux-amd64 .
	# macOS版本
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(PROJECT_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(PROJECT_NAME)-darwin-arm64 .

test: ## 运行测试
	go test -v ./...

install-deps: ## 安装依赖
	go mod tidy