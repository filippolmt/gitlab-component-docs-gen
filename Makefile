BINARY := gitlab-component-docs-gen

.PHONY: build test clean

build:
	go build -o $(BINARY) main.go

test:
	go test -v ./...

clean:
	rm -f $(BINARY)
