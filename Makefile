PROJECT_NAME := yt-dlp-simpgo
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

LDFLAGS := -X 'main.Version=$(VERSION)' -s -w
DIST := dist

.PHONY: build build-gui clean clean-runtime clean-all release test help

help: ## 显示帮助信息
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 构建项目
	mkdir -p $(DIST)
	go build -ldflags="$(LDFLAGS)" -o $(DIST)/$(PROJECT_NAME) .

build-gui: ## 构建Windows GUI版本
	mkdir -p $(DIST)
	rm -f rsrc.syso rsrc_windows_*.syso
	go run github.com/akavel/rsrc@v0.10.2 -ico res/icon.ico -arch amd64 -o rsrc_windows_amd64.syso
	GOOS=windows go build -ldflags="$(LDFLAGS) -H=windowsgui" -o $(DIST)/$(PROJECT_NAME)-windows-gui.exe .

clean: ## 清理构建产物
	rm -rf $(DIST)
	rm -f rsrc.syso rsrc_windows_*.syso

clean-runtime: ## 清理运行时文件
	rm -f yt-dlp-simpgo.ini yt-dlp.conf yt-dlp yt-dlp.exe .yt-dlp-simpgo-update-* .yt-dlp-simpgo-update.ps1
	rm -rf 下载

clean-all: clean clean-runtime ## 清理构建产物和运行时文件

release: clean ## 构建发布版本
	mkdir -p $(DIST)
	rm -f rsrc.syso rsrc_windows_*.syso
	go run github.com/akavel/rsrc@v0.10.2 -ico res/icon.ico -arch amd64 -o rsrc_windows_amd64.syso
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS) -H=windowsgui" -o $(DIST)/$(PROJECT_NAME)-windows-amd64.exe .
	rm -f rsrc_windows_*.syso
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST)/$(PROJECT_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST)/$(PROJECT_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST)/$(PROJECT_NAME)-darwin-arm64 .

test: ## 运行测试
	go test -v ./...

install-deps: ## 安装依赖
	go mod tidy
