BINARY=felituive

build:
	go build -o $(BINARY) ./cmd/felituive

run: build
	./$(BINARY)

install: build
	mv $(BINARY) /usr/local/bin/

clean:
	rm -f $(BINARY)
	rm -rf dist/
