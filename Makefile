.PHONY: build test check update clean demo

build:
	mkdir -p bin
	go build -trimpath -o bin/saveme ./cmd/saveme
	go build -trimpath -o bin/saved ./cmd/saved
	go build -trimpath -o bin/saveme-update ./cmd/saveme-update

test:
	go test -race ./...
	bash scripts/check_test.sh

check:
	go vet ./...

demo: build
	./bin/saved -demo 2>bin/demo.log & pid=$$!; trap 'kill $$pid' EXIT; \
	until ./bin/saveme -demo -status >/dev/null 2>&1; do sleep 0.1; done; \
	./bin/saveme -demo

update:
	$(MAKE) test
	$(MAKE) check
	$(MAKE) build
	sudo bash deploy/update.sh "$$(id -un)"

clean:
	rm -f bin/saveme bin/saved bin/saveme-update
