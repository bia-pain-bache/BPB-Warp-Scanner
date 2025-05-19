VER ?= $(VERSION)
APP_NAME := BPB-Warp-Scanner
GO := GO111MODULE=on CGO_ENABLED=0 go
LDFLAGS = -w -s \
    -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
    -X "main.VERSION=$(VER)" \
    -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

GOARCH_DARWIN := amd64 arm64
GOARCH_LINUX := $(GOARCH_DARWIN) 386 arm
GOARCH_WINDOWS := $(GOARCH_DARWIN) 386 arm

OUT_DIR := bin
DIST_DIR := dist
CORE_DIR := core
XRAY_RELEASE_URL := https://github.com/bia-pain-bache/Xray-core/releases/latest/download
XRAY_ZIP := temp/xray.zip


.PHONY: build clean

build: pre-build linux windows darwin

pre-build:
	@mkdir -p $(OUT_DIR) $(DIST_DIR) temp

linux:
	@for arch in $(GOARCH_LINUX); do \
		echo "Downloading latest Xray-core release..."; \
		curl -L -o $(XRAY_ZIP) $(XRAY_RELEASE_URL)/Xray-linux-$$arch.zip; \
		unzip -o $(XRAY_ZIP) -d $(CORE_DIR); \
		rm $(XRAY_ZIP); \
		echo "Building for linux-$$arch..."; \
		outdir="$(OUT_DIR)/$(APP_NAME)-linux-$$arch"; \
		mkdir -p "$$outdir"; \
		outfile="$(APP_NAME)-linux-$$arch"; \
		GOOS=linux GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$$outfile"; \
		cp LICENSE "$$outdir/"; \
		cp -r $(CORE_DIR) "$$outdir/"; \
		tar -cvzf "$(DIST_DIR)/$$outfile.tar.gz" -C "$$outdir/" .; \
	done
	rm "$(CORE_DIR)/xray"

windows:
	@for arch in $(GOARCH_WINDOWS); do \
		echo "Downloading latest Xray-core release..."; \
		curl -L -o $(XRAY_ZIP) $(XRAY_RELEASE_URL)/Xray-windows-$$arch.zip; \
		unzip -o $(XRAY_ZIP) -d $(CORE_DIR); \
		rm $(XRAY_ZIP); \
		echo "Building for windows-$$arch..."; \
		outdir="$(OUT_DIR)/$(APP_NAME)-windows-$$arch"; \
		mkdir -p "$$outdir"; \
		outfile="$(APP_NAME)-windows-$$arch.exe"; \
		GOOS=windows GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$$outfile"; \
		cp LICENSE "$$outdir/"; \
		cp -r $(CORE_DIR) "$$outdir/"; \
		(cd "$$outdir" && zip -9vr -q "../../$(DIST_DIR)/BPB-Warp-Scanner-windows-$$arch.zip" .); \
	done
	rm "$(CORE_DIR)/xray.exe"

darwin:
	@for arch in $(GOARCH_DARWIN); do \
		echo "Downloading latest Xray-core release..."; \
		curl -L -o $(XRAY_ZIP) $(XRAY_RELEASE_URL)/Xray-darwin-$$arch.zip; \
		unzip -o $(XRAY_ZIP) -d $(CORE_DIR); \
		rm $(XRAY_ZIP); \
		echo "Building for darwin-$$arch..."; \
		outdir="$(OUT_DIR)/$(APP_NAME)-darwin-$$arch"; \
		mkdir -p "$$outdir"; \
		outfile="$(APP_NAME)-darwin-$$arch"; \
		GOOS=darwin GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$$outfile"; \
		cp LICENSE "$$outdir/"; \
		cp -r $(CORE_DIR) "$$outdir/"; \
		(cd "$$outdir" && zip -9vr -q "../../$(DIST_DIR)/BPB-Warp-Scanner-darwin-$$arch.zip" .); \
	done
	rm "$(CORE_DIR)/xray" "$(CORE_DIR)/LICENSE" 

clean:
	@rm -rf $(OUT_DIR) $(DIST_DIR)
