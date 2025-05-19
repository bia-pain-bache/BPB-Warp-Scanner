VER ?= $(VERSION)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
APP_NAME := BPB-Warp-Scanner
GO := GO111MODULE=on CGO_ENABLED=0 go
LDFLAGS = -w -s \
    -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
    -X "main.VERSION=$(VER)" \
    -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

OUT_DIR := bin
DIST_DIR := dist
CORE_DIR := core
XRAY_RELEASE_URL := https://github.com/bia-pain-bache/Xray-core/releases/latest/download
XRAY_ZIP := temp/xray.zip


.PHONY: build clean

pre-build:
	@mkdir -p $(OUT_DIR) $(DIST_DIR) temp

build: pre-build
	@echo "Downloading latest Xray-core release..."; \
	curl -L -o $(XRAY_ZIP) $(XRAY_RELEASE_URL)/Xray-$(GOOS)-$(GOARCH).zip; \
	unzip -o $(XRAY_ZIP) -d $(CORE_DIR); \
	rm $(XRAY_ZIP); \
	echo "Building for $(GOOS)-$(GOARCH)..."; \
	outdir="$(OUT_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH)"; \
	mkdir -p "$$outdir"; \
	outfile="$(APP_NAME)-$(GOOS)-$(GOARCH)"; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o "$$outdir/$$outfile"; \
	cp LICENSE "$$outdir/"; \
	cp -r $(CORE_DIR) "$$outdir/"; \
	if [ "$(GOOS)" = "windows" ]; then \
		(cd "$$outdir" && zip -9vr -q "../../$(DIST_DIR)/BPB-Warp-Scanner-$(GOOS)-$(GOARCH).zip" .); \
	else \
		tar -cvzf "$(DIST_DIR)/$$outfile.tar.gz" -C "$$outdir/" .; \
	fi; \
	rm -f $(CORE_DIR)/xray*

clean:
	@rm -rf $(OUT_DIR) $(DIST_DIR)
	@rm -f $(CORE_DIR)/xray*
	@rm -f $(CORE_DIR)/LICENSE
