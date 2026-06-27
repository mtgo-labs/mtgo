.PHONY: all fmt fmt-check tidy vet lint test test-race bench bench-compare bench-baseline clean

# Benchmark packages (kept in sync with .github/workflows/benchmark.yml)
BENCH_PKGS  := ./internal/crypto/ ./internal/session/ ./internal/transport/ ./tg/ ./tgerr/ ./telegram/fileid/ ./telegram/types/ ./telegram/parser/
BENCH_FLAGS := -run=xxx -bench=. -benchmem -benchtime=1s
BENCH_COUNT ?= 5
REF        ?= main

# ---- Targets ----

## all: fmt + vet + lint + test (the pre-commit gate)
all: fmt vet lint test

## fmt: format all Go files in place
fmt:
	gofmt -w .

## fmt-check: fail if any file is unformatted (CI)
fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "Files need formatting:"; gofmt -l .; exit 1; }

## tidy: go mod tidy
tidy:
	go mod tidy

## vet: go vet ./...
vet:
	go vet ./...

## lint: golangci-lint run
lint:
	golangci-lint run

## test: go test ./...
test:
	go test ./...

## test-race: go test -race ./...
test-race:
	go test -race ./...

## bench: run benchmarks on the current tree
bench:
	go test $(BENCH_FLAGS) -count=$(BENCH_COUNT) $(BENCH_PKGS)

## bench-compare: compare current HEAD against REF (default: main)
##   Usage: make bench-compare REF=main
##          make bench-compare REF=v0.9.0
##          make bench-compare REF=HEAD~3
bench-compare: _benchstat
	@echo "==> Benchmarking current HEAD → new.txt"
	@go test $(BENCH_FLAGS) -count=$(BENCH_COUNT) $(BENCH_PKGS) 2>&1 | tee new.txt
	@echo ""
	@echo "==> Benchmarking $(REF) → old.txt (detached sibling worktree)"
	@set -e; \
	WT=$$(TMPDIR="$$(dirname "$(CURDIR)")" mktemp -d); \
	trap 'git worktree remove --force $$WT 2>/dev/null; rm -rf $$WT' EXIT; \
	git worktree add --detach -q "$$WT" "$(REF)"; \
	( cd "$$WT" && GOWORK=off go test $(BENCH_FLAGS) -count=$(BENCH_COUNT) $(BENCH_PKGS) ) 2>&1 | tee old.txt
	@echo ""
	@echo "==> benchstat old.txt new.txt"
	@benchstat old.txt new.txt

## bench-baseline: regenerate the committed bench_baseline.txt (count=10)
##   Run locally on stable hardware, then commit the result.
##   CI compares current-commit results against this file.
bench-baseline:
	go test $(BENCH_FLAGS) -count=10 $(BENCH_PKGS) > bench_baseline.txt 2>&1
	@echo "==> Wrote bench_baseline.txt"
	@benchstat bench_baseline.txt | head -5

## clean: remove benchmark artifacts
clean:
	rm -f old.txt new.txt bench_result.txt

# ---- Internal helpers ----

.PHONY: _benchstat
_benchstat:
	@command -v benchstat >/dev/null 2>&1 || go install golang.org/x/perf/cmd/benchstat@latest
