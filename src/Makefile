FILE=./*.go
ARCH=amd64
OS=linux
FILENAME=stickerDownloader
VERSION=1.0

# 确保 /bin 文件夹存在
ensure_bin_dir:
	@mkdir -p ../bin

build: ensure_bin_dir
	@echo "正在编译$(ARCH)架构$(OS)平台的文件"
	GOARCH=$(ARCH) GOOS=$(OS) go build -o ../bin/$(FILENAME)-$(ARCH)-linux-$(VERSION) $(FILE)
	@echo "文件$(FILENAME)-$(ARCH)-linux-$(VERSION) 编译完成"

run: 
	go run ./*.go

build_all: ensure_bin_dir
	@echo "正在编译全平台的文件"
	GOARCH=amd64 GOOS=linux go build -o ../bin/$(FILENAME)-amd64-linux-$(VERSION) $(FILE)
	GOARCH=arm64 GOOS=linux go build -o ../bin/$(FILENAME)-arm64-linux-$(VERSION) $(FILE)
	GOARCH=amd64 GOOS=windows go build -o ../bin/$(FILENAME)-amd64-windows-$(VERSION) $(FILE)
	GOARCH=arm64 GOOS=windows go build -o ../bin/$(FILENAME)-arm64-windows-$(VERSION) $(FILE)

test: ensure_bin_dir
	@echo "正在编译全平台的文件"
	GOARCH=amd64 GOOS=linux go build -o ../bin/$(FILENAME)-amd64-linux-$(VERSION)-test $(FILE)
	GOARCH=arm64 GOOS=linux go build -o ../bin/$(FILENAME)-arm64-linux-$(VERSION)-test $(FILE)
	GOARCH=amd64 GOOS=windows go build -o ../bin/$(FILENAME)-amd64-windows-$(VERSION)-test $(FILE)
	GOARCH=arm64 GOOS=windows go build -o ../bin/$(FILENAME)-arm64-windows-$(VERSION)-test $(FILE)
	@sleep 3s
	@rm ../bin/*test
	@echo "构筑文件清理完成"