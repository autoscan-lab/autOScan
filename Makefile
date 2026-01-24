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
	go build -o $(DIST)/$(BINARY)-darwin-arm64 ./cmd/autoscan
	docker build -q --platform linux/amd64 -o $(DIST) .
	@ls -lh $(DIST)/

.PHONY: build install uninstall clean release
