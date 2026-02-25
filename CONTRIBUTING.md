# Contributing

Thanks for contributing to `gen-dto`.

## Prerequisites

- Go `1.26.x`
- Git

## Local Setup

```bash
git clone https://github.com/seitarof/gen-dto.git
cd gen-dto
go mod tidy
```

## Development Workflow

1. Create a feature branch.
2. Implement changes with tests.
3. Run checks locally.
4. Open a pull request.

## Required Local Checks

```bash
# format
find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 gofmt -w

# static checks
go vet ./...

# tests
go test ./...
```

## Benchmark Check (Optional but Recommended)

```bash
go test -run '^$' -bench . -benchmem \
  ./internal/cli \
  ./internal/parser \
  ./internal/resolver \
  ./internal/generator
```

## Pull Request Guidelines

- Keep PRs focused and small.
- Add/adjust tests for behavior changes.
- Keep generated output deterministic.
- Update README when user-facing behavior changes.

## Commit Style

Conventional-style prefixes are recommended:

- `feat:` new functionality
- `fix:` bug fixes
- `test:` tests
- `docs:` documentation
- `chore:` maintenance

## Release

Releases are automated by GitHub Actions + GoReleaser on `v*` tags.

- Workflow: `.github/workflows/release.yml`
- Config: `.goreleaser.yaml`

For Homebrew publication, the repository secret below must be set:

- `HOMEBREW_TAP_GITHUB_TOKEN`
