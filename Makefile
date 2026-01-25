BINARY=autoscan
INSTALL_PATH=$(HOME)/.local/bin/$(BINARY)
CONFIG_DIR=$(HOME)/.config/autoscan
DIST=dist

build:
	go build -o $(BINARY) ./cmd/autoscan

install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(INSTALL_PATH)
	@echo "Installed to $(INSTALL_PATH)"

uninstall:
	rm -f $(INSTALL_PATH)
	rm -rf $(CONFIG_DIR)

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

release: clean
	@mkdir -p $(DIST)
	@echo "Building macOS arm64..."
	go build -o $(DIST)/$(BINARY)-darwin-arm64 ./cmd/autoscan
	@echo "Building Linux amd64 via Docker..."
	docker build -q --platform linux/amd64 -o $(DIST) .
	@echo ""
	@ls -lh $(DIST)/

windows:
	@mkdir -p $(DIST)
	@echo "Building Windows amd64..."
	go build -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/autoscan
	@echo ""
	@ls -lh $(DIST)/

.PHONY: build install uninstall clean release windows
