VER ?= $(VERSION)
APP_NAME := BPB-Warp-Scanner
GO := GO111MODULE=on CGO_ENABLED=0 go
LDFLAGS = -w -s \
    -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
    -X "main.VERSION=$(VER)" \
    -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

GOARCH_LIST := amd64 386
GOARCH_LINUX := $(GOARCH_LIST) arm64 arm
GOARCH_WINDOWS := $(GOARCH_LIST)

OUT_DIR := bin
DIST_DIR := dist

.PHONY: build clean

build: pre-build linux windows

pre-build:
	@mkdir -p $(OUT_DIR) $(DIST_DIR)

linux:
	@for arch in $(GOARCH_LINUX); do \
		echo "Building for linux-$$arch..."; \
		outdir="$(OUT_DIR)/$(APP_NAME)-linux-$$arch"; \
		outfile="$(APP_NAME)-linux-$$arch"; \
		GOOS=linux GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$$outfile"; \
		cp LICENSE "$$outdir/"; \
		tar -czf "$(DIST_DIR)/$$outfile.tar.gz" -C "$$outdir/" .; \
	done

windows:
	@for arch in $(GOARCH_WINDOWS); do \
		echo "Building for windows-$$arch..."; \
		outdir="$(OUT_DIR)/$(APP_NAME)-windows-$$arch"; \
		outfile="$(APP_NAME)-windows-$$arch.exe"; \
		GOOS=windows GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$$outfile"; \
		cp LICENSE "$$outdir/"; \
		zip -j -q "$(DIST_DIR)/BPB-Warp-Scanner-windows-$$arch.zip" "$$outdir/"*; \
	done

clean:
	@rm -rf $(OUT_DIR) $(DIST_DIR)
