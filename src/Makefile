FILE=./*.go
ARCH=amd64
OS=linux
FILENAME=stickerDownloader
VERSION=1.0

VERSION = $(shell git describe --tags --abbrev=0)
COMMIT_HASH = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date --iso-8601=seconds)
LDFLAGS = -X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)

# 确保 /bin 文件夹存在
ensure_bin_dir:
	@mkdir -p ../bin

build: ensure_bin_dir
	@echo "正在编译$(ARCH)架构$(OS)平台的文件"
	GOARCH=$(ARCH) GOOS=$(OS) go build -o ../bin/$(FILENAME)-$(ARCH)-$(OS)-$(VERSION) \
		-ldflags "$(LDFLAGS)" $(FILE)
	cp ../bin/$(FILENAME)-$(ARCH)-$(OS)-$(VERSION) ../bin/$(FILENAME)-$(ARCH)-$(OS)-$(VERSION)-upx
	upx --best --lzma ../bin/$(FILENAME)-$(ARCH)-$(OS)-$(VERSION)-upx
	@echo "编译完成: $(FILENAME)-$(ARCH)-$(OS)-$(VERSION)"

run:
	go run . --config ../config/config.yaml

unit_test:
	go test ./...

build_all: ensure_bin_dir
	@echo "正在编译全平台的文件"
	GOARCH=amd64 GOOS=linux go build -o ../bin/$(FILENAME)-amd64-linux-$(VERSION) \
		-ldflags "$(LDFLAGS)" $(FILE)
	GOARCH=arm64 GOOS=linux go build -o ../bin/$(FILENAME)-arm64-linux-$(VERSION) \
		-ldflags "$(LDFLAGS)" $(FILE)
	GOARCH=amd64 GOOS=windows go build -o ../bin/$(FILENAME)-amd64-windows-$(VERSION).exe \
		-ldflags "$(LDFLAGS)" $(FILE)
	GOARCH=arm64 GOOS=windows go build -o ../bin/$(FILENAME)-arm64-windows-$(VERSION).exe \
		-ldflags "$(LDFLAGS)" $(FILE)
	@echo "正在生成 UPX 压缩版本"
	cp ../bin/$(FILENAME)-amd64-linux-$(VERSION) ../bin/$(FILENAME)-amd64-linux-$(VERSION)-upx
	cp ../bin/$(FILENAME)-arm64-linux-$(VERSION) ../bin/$(FILENAME)-arm64-linux-$(VERSION)-upx
	cp ../bin/$(FILENAME)-amd64-windows-$(VERSION).exe ../bin/$(FILENAME)-amd64-windows-$(VERSION)-upx.exe
	cp ../bin/$(FILENAME)-arm64-windows-$(VERSION).exe ../bin/$(FILENAME)-arm64-windows-$(VERSION)-upx.exe
	upx --best --lzma ../bin/$(FILENAME)-amd64-linux-$(VERSION)-upx
	upx --best --lzma ../bin/$(FILENAME)-arm64-linux-$(VERSION)-upx
	upx --best --lzma ../bin/$(FILENAME)-amd64-windows-$(VERSION)-upx.exe
	-upx --best --lzma ../bin/$(FILENAME)-arm64-windows-$(VERSION)-upx.exe || true
	@echo "全平台编译完成"

test: ensure_bin_dir
	@echo "正在编译全平台的文件"
	GOARCH=amd64 GOOS=linux go build -o ../bin/$(FILENAME)-amd64-linux-$(VERSION)-test $(FILE)
	GOARCH=arm64 GOOS=linux go build -o ../bin/$(FILENAME)-arm64-linux-$(VERSION)-test $(FILE)
	GOARCH=amd64 GOOS=windows go build -o ../bin/$(FILENAME)-amd64-windows-$(VERSION)-test.exe $(FILE)
	GOARCH=arm64 GOOS=windows go build -o ../bin/$(FILENAME)-arm64-windows-$(VERSION)-test.exe $(FILE)
	@sleep 3s
	@rm ../bin/*test*
	@echo "构筑文件清理完成"
