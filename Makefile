BINARY_NAME=syncpaper
CMD_DIR=./cmd/syncpaper
PREFIX?=/usr/local

.PHONY: all build test clean install uninstall package dist

all: build

build:
	@echo "==> Building $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(CMD_DIR)
	@echo "==> Built $(BINARY_NAME)"

test:
	@echo "==> Running test suite..."
	go test -v ./tests/...

dist: package

package:
	@echo "==> Building release packages for Linux (amd64, arm64)..."
	@mkdir -p dist/packages
	@for arch in amd64 arm64; do \
		mkdir -p dist/$(BINARY_NAME)-linux-$$arch; \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -ldflags="-s -w" -o dist/$(BINARY_NAME)-linux-$$arch/$(BINARY_NAME) $(CMD_DIR); \
		cp README.md dist/$(BINARY_NAME)-linux-$$arch/; \
		tar -czf dist/packages/$(BINARY_NAME)-linux-$$arch.tar.gz -C dist $(BINARY_NAME)-linux-$$arch; \
		rm -rf dist/$(BINARY_NAME)-linux-$$arch; \
	done
	@cd dist/packages && sha256sum $(BINARY_NAME)-*.tar.gz > SHA256SUMS
	@echo "==> Packages created in dist/packages/:"
	@ls -la dist/packages/

install: build
	@echo "==> Installing to $(PREFIX)/bin/$(BINARY_NAME)..."
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY_NAME) $(PREFIX)/bin/$(BINARY_NAME)
	@echo "==> Installed successfully."

uninstall:
	@echo "==> Removing $(PREFIX)/bin/$(BINARY_NAME)..."
	rm -f $(PREFIX)/bin/$(BINARY_NAME)
	@echo "==> Uninstalled."

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BINARY_NAME) dist

