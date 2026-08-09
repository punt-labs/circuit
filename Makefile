STATICCHECK ?= staticcheck
MARKDOWNLINT ?= markdownlint-cli2
NPM ?= npm
PROBCLI ?= probcli

.PHONY: check check-engine check-pi-extension check-docs check-machines lint lint-engine lint-pi-extension docs test test-engine build build-engine typecheck-pi-extension format-check-pi-extension clean

check: check-engine check-pi-extension check-docs

check-machines:
	$(PROBCLI) machines/build-job.mch -init
	$(PROBCLI) machines/build-job.mch -model_check -nodead
	$(PROBCLI) machines/pr-watch.mch -init
	$(PROBCLI) machines/pr-watch.mch -model_check -nodead
	$(PROBCLI) machines/review-flow.mch -init
	$(PROBCLI) machines/review-flow.mch -model_check -nodead

check-engine: lint-engine test-engine

lint: lint-engine

lint-engine:
	gofmt -w ./cmd ./internal
	go vet ./...
	$(STATICCHECK) ./...

test: test-engine

test-engine:
	go test -race -count=1 ./...

build: build-engine

build-engine:
	go build -o circuit ./cmd/circuit

check-pi-extension: typecheck-pi-extension lint-pi-extension format-check-pi-extension

typecheck-pi-extension:
	$(NPM) --prefix .pi run typecheck

lint-pi-extension:
	$(NPM) --prefix .pi run lint

format-check-pi-extension:
	$(NPM) --prefix .pi run format:check

docs: check-docs

check-docs:
	$(MARKDOWNLINT) "**/*.md" "#node_modules" "#.pi/node_modules" "#.tmp"

clean:
	rm -f circuit coverage.out
	rm -rf dist
