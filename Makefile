NAME = stickerDownloader

VERSION = $(shell git describe --tags --abbrev=0)
COMMIT_HASH = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date --iso-8601=seconds)

.PHONY: build_all build build_internal clean
build:
	OS=$(shell uname -s) ARCH=$(shell uname -m) GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') $(MAKE) build_internal

build_all:
	GOOS=windows GOARCH=amd64 OS=Windows ARCH=x86_64 $(MAKE) build_internal
	GOOS=windows GOARCH=arm64 OS=Windows ARCH=aarch64 $(MAKE) build_internal
	GOOS=linux GOARCH=amd64 OS=Linux ARCH=x86_64 $(MAKE) build_internal
	GOOS=linux GOARCH=arm64 OS=Linux ARCH=aarch64 $(MAKE) build_internal

build_internal:
	@echo "正在构建 $(OS)-$(ARCH) 下的 $(NAME)-$(VERSION)-$(COMMIT_HASH)"
	cd src; GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../bin/$(GOOS)_$(GOARCH)/$(NAME)-$(VERSION)-$(COMMIT_HASH) \
		-ldflags "-X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" \
		./main.go

clean:
	rm -rf bin

run: build
	bin/$(shell uname -s | tr A-Z a-z)_$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')/$(NAME)-$(VERSION)-$(COMMIT_HASH)
