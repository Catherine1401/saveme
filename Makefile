.PHONY: build test check clean

build:
	mkdir -p bin
	go build -trimpath -o bin/saveme ./cmd/saveme
	go build -trimpath -o bin/saved ./cmd/saved

test:
	go test -race ./...

check:
	go vet ./...

clean:
	rm -f bin/saveme bin/saved
