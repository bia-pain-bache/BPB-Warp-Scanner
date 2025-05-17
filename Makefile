VER ?= $(VERSION)
LDFLAGS = -w -s \
		  -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
          -X "main.VERSION=$(VER)" \
          -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

GO := GO111MODULE=on CGO_ENABLED=0 go
GOLANGCI_LINT_VERSION = v1.61.0

GOOS_LIST := linux windows
GOARCH_LIST := amd64 arm64 arm 386

.PHONY: build
build:
	@mkdir -p bin dist
	@for goos in $(GOOS_LIST); do \
		if [ "$$goos" = "windows" ]; then \
			ext=".exe"; \
		else \
			ext=""; \
		fi; \
		for goarch in $(GOARCH_LIST); do \
			if [ "$$goos" = "windows" && ("$$goarch" = "arm64" || "$$goarch" = "arm")]; then \
				continue: \
			fi; \
			echo "Building for $$goos-$$goarch..."; \
			outdir="bin/BPB-Warp-Scanner-$$goos-$$goarch"; \
			outfile="BPB-Warp-Scanner-$$goos-$$goarch$$ext"; \
			GOOS=$$goos GOARCH=$$goarch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $$outdir/$$outfile; \
			cp LICENSE $$outdir/; \
			if [ "$$goos" = "windows" ]; then \
                zipfile="dist/BPB-Warp-Scanner-$$goos-$$goarch.zip"; \
                echo "Zipping $$outfile -> $$zipfile"; \
                zip -j -q $$zipfile $$outdir/*; \
            else \
                tarfile="dist/BPB-Warp-Scanner-$$goos-$$goarch.tar.gz"; \
                echo "Zipping $$outfile -> $$tarfile"; \
                tar -czf $$tarfile -C $$outdir/ .; \
            fi; \
		done; \
	done
