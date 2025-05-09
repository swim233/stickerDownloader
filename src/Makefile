FILE=./*.go
ARCH=amd64
OS=linux
FILENAME=stickerDownloader
VERSION=1.0

VERSION = $(shell git describe --tags --abbrev=0)
COMMIT_HASH = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date --iso-8601=seconds)

# 确保 /bin 文件夹存在
ensure_bin_dir:
	@mkdir -p ../bin

build: ensure_bin_dir
	@echo "正在编译$(ARCH)架构$(OS)平台的文件"
	GOARCH=$(ARCH) GOOS=$(OS) go build -o ../bin/$(FILENAME)-$(ARCH)-linux-$(VERSION) \
		-ldflags "-X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" \
		$(FILE)
	@echo "文件$(FILENAME)-$(ARCH)-linux-$(VERSION) 编译完成"

run: 
	go run ./*.go

build_all: ensure_bin_dir
	@echo "正在编译全平台的文件"
	GOARCH=amd64 GOOS=linux go build -o ../bin/$(FILENAME)-amd64-linux-$(VERSION) \
		-ldflags "-X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" \
		$(FILE)
	GOARCH=arm64 GOOS=linux go build -o ../bin/$(FILENAME)-arm64-linux-$(VERSION) \
		-ldflags "-X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" \
		$(FILE)
	GOARCH=amd64 GOOS=windows go build -o ../bin/$(FILENAME)-amd64-windows-$(VERSION).exe \
		-ldflags "-X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" \
		$(FILE)
	GOARCH=arm64 GOOS=windows go build -o ../bin/$(FILENAME)-arm64-windows-$(VERSION).exe \
		-ldflags "-X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" \
		$(FILE)

test: ensure_bin_dir
	@echo "正在编译全平台的文件"
	GOARCH=amd64 GOOS=linux go build -o ../bin/$(FILENAME)-amd64-linux-$(VERSION)-test $(FILE)
	GOARCH=arm64 GOOS=linux go build -o ../bin/$(FILENAME)-arm64-linux-$(VERSION)-test $(FILE)
	GOARCH=amd64 GOOS=windows go build -o ../bin/$(FILENAME)-amd64-windows-$(VERSION)-test.exe $(FILE)
	GOARCH=arm64 GOOS=windows go build -o ../bin/$(FILENAME)-arm64-windows-$(VERSION)-test.exe $(FILE)
	@sleep 3s
	@rm ../bin/*test
	@echo "构筑文件清理完成"
