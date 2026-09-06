BINARY_NAME=syncpaper
CMD_DIR=./cmd/syncpaper
PREFIX?=/usr/local

.PHONY: all build test clean install uninstall

all: build

build:
	@echo "==> Building $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(CMD_DIR)
	@echo "==> Built $(BINARY_NAME)"

test:
	@echo "==> Running test suite..."
	go test -v ./tests/...

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
	rm -f $(BINARY_NAME)
