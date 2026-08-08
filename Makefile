STATICCHECK ?= staticcheck
MARKDOWNLINT ?= markdownlint-cli2

.PHONY: check lint docs test build clean

check: lint docs test

lint:
	gofmt -w ./cmd
	go vet ./...
	$(STATICCHECK) ./...

docs:
	$(MARKDOWNLINT) "**/*.md" "#node_modules"

test:
	go test -race -count=1 ./...

build:
	go build -o circuit ./cmd/circuit

clean:
	rm -f circuit coverage.out
	rm -rf dist
