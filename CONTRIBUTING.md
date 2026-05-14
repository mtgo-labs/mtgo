# Contributing to mtgo

Thank you for your interest in contributing! This document outlines the process and guidelines for contributing to mtgo.

## Prerequisites

- **Go 1.22+** — check with `go version`
- **Git** — for version control

## Getting Started

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/mtgo.git
   cd mtgo
   ```
3. **Add upstream** remote:
   ```bash
   git remote add upstream https://github.com/mtgo-labs/mtgo.git
   ```

## Development Workflow

### Branching

- Create a branch from `main` for each piece of work:
  ```bash
  git checkout -b feat/short-description
  ```
- Use conventional prefixes:
  - `feat/` — new features
  - `fix/` — bug fixes
  - `refactor/` — code restructuring
  - `docs/` — documentation changes
  - `test/` — test additions or fixes
  - `chore/` — maintenance tasks

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
```

Run tests for a specific package:

```bash
go test ./telegram/...
go test ./tg/...
```

### Code Generation

The `tg/` package contains generated code from TL schemas. If you modify the compiler or schema:

```bash
go run cmd/tlgen/main.go
```

This regenerates all `_gen.go` files under `tg/`. **Do not edit generated files directly.**

### Error Generation

Error types in `tgerr/` are generated from the schema:

```bash
go run cmd/errgen/main.go
```

## Code Style

- Follow standard Go conventions: [Effective Go](https://go.dev/doc/effective_go)
- Run `go vet` before submitting:
  ```bash
  go vet ./...
  ```
- Format your code:
  ```bash
  gofmt -w .
  ```
- Use `gofmt`-style formatting; no custom formatters required.

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): short description

Optional longer body explaining the change.
```

**Types:** `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`

**Scopes:** `tg`, `telegram`, `tgerr`, `session`, `transport`, `storage`, `crypto`, `compiler`, `parser`, `fileid`, `client`

**Examples:**

```
feat(telegram): add inline mode support
fix(session): resolve reconnection loop on network change
docs(tg): update API reference for layer 192
chore(deps): bump modernc.org/sqlite to v1.50.0
```

## Pull Request Process

1. **Rebase** on upstream `main` before opening:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```
2. **Ensure** all tests pass and the build is clean:
   ```bash
   go build ./...
   go test ./...
   go vet ./...
   ```
3. **Write** a clear PR description explaining:
   - What changed and why
   - Any breaking changes
   - Related issues (e.g., `Fixes #12`)
4. **Keep PRs focused** — one logical change per PR.
5. **Add tests** for new functionality or bug fixes.

## Reporting Issues

- Use [GitHub Issues](https://github.com/mtgo-labs/mtgo/issues).
- Include:
  - Go version (`go version`)
  - OS and architecture
  - Minimal reproducible example
  - Expected vs. actual behavior
  - Relevant logs (redact any sensitive data)

## Project Structure

```
.
├── cmd/
│   ├── tlgen/        # TL schema compiler
│   └── errgen/       # Error type generator
├── compiler/
│   └── tlgen/        # TL compiler implementation and templates
├── docs/
│   └── API_REFERENCE.md
├── examples/
│   └── echo_bot/     # Example bot
├── internal/
│   ├── crypto/       # MTProto crypto primitives
│   ├── session/      # MTProto session management
│   ├── storage/      # Session persistence backends
│   └── transport/    # TCP/ABR transport layer
├── storage/          # Public storage interfaces
├── tg/               # Generated TL types (do not edit directly)
├── tgerr/            # Generated error types
└── telegram/         # High-level Client API
```

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).