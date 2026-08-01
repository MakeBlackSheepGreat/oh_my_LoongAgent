# Makefile - 多平台多架构构建
# 支持 Windows/Linux/macOS × amd64/arm64/loong64/riscv64
# CGO_ENABLED=0 单静态二进制

GO ?= go
CGO_ENABLED := 0
LDFLAGS := -s -w
BINARY := slim-agent
BUILD_DIR := dist

# 目标平台与架构组合
PLATFORMS := \
    windows/amd64 \
    windows/arm64 \
    linux/amd64 \
    linux/arm64 \
    linux/loong64 \
    linux/riscv64 \
    darwin/amd64 \
    darwin/arm64

.PHONY: all build test vet clean cross-compile frontend

all: frontend build

# 本平台构建
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/server/

# 前端构建
frontend:
	cd web && npm run build

# 测试
test:
	$(GO) test -v -count=1 ./...

# vet
vet:
	$(GO) vet ./...

# 多平台多架构交叉编译
cross-compile: $(PLATFORMS)

$(PLATFORMS):
	@echo "Building $@..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) CGO_ENABLED=$(CGO_ENABLED) \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@))$(if $(findstring windows,$(word 1,$(subst /, ,$@))),.exe,) ./cmd/server/

# 清理
clean:
	rm -rf $(BUILD_DIR)
	rm -rf web/dist
