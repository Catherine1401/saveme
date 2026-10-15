.PHONY: build test check clean demo

build:
	mkdir -p bin
	go build -trimpath -o bin/saveme ./cmd/saveme
	go build -trimpath -o bin/saved ./cmd/saved

test:
	go test -race ./...

check:
	go vet ./...

demo: build
	./bin/saved -demo 2>bin/demo.log & pid=$$!; trap 'kill $$pid' EXIT; \
	until ./bin/saveme -demo -status >/dev/null 2>&1; do sleep 0.1; done; \
	./bin/saveme -demo

clean:
	rm -f bin/saveme bin/saved
