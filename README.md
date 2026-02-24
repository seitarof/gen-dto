# gen-dto

`gen-dto` is a Go CLI that generates DTO conversion code between structs via `go:generate`.

It is designed for reducing repetitive mapping code and currently generates **both directions in one file**:

- `A -> B`
- `B -> A`

## Features

- Generate conversion functions from source/destination struct types
- Case-insensitive field matching
- `--ignore-fields` support
- Type alias field support (for example `type X = otherpkg.Y`)
- Recursive nested struct conversion generation (in the same package path)
- Built-in conversion rules:
  - Same type assignment
  - Basic casts (for convertible basic types)
  - Pointer wrap/unwrap
  - `database/sql.Null*` <-> value conversions
  - `time.Time` <-> `string` conversions (`RFC3339`)
  - Slice element conversion
  - `Stringer` to `string`
  - Assignable/Convertible fallback
- Import formatting via `goimports`

## Supported Go Version

- **Go 1.26.x**

The repository tracks Go via `go.mod` and CI uses `actions/setup-go` with `go-version-file: go.mod` and `check-latest: true`.

## Installation

```bash
go install github.com/seitarof/gen-dto/cmd/gen-dto@latest
```

Homebrew:

```bash
brew tap seitarof/homebrew-tap
brew install gen-dto
```

## Quick Start

Add `go:generate` to your code:

```go
//go:generate gen-dto \
//  --src-type=User \
//  --src-path=./internal/domain/model \
//  --dst-type=UserResponse \
//  --dst-path=./internal/dto \
//  --filename=user_conv_gen.go \
//  --ignore-fields=Password,SecretKey
```

Run:

```bash
go generate ./...
```

A single output file will include both directions, for example:

- `ConvertUserToUserResponse`
- `ConvertUserResponseToUser`

If source/destination type names are the same but package paths differ, generated function names include package tokens to avoid collisions, for example:

- `ConvertSourceAddressToDestAddress`
- `ConvertDestAddressToSourceAddress`

## CLI Options

Required:

- `--src-type`, `-s`: source struct type name
- `--src-path`: source package path
- `--dst-type`, `-d`: destination struct type name
- `--dst-path`: destination package path
- `--filename`, `-o`: output file path

Optional:

- `--ignore-fields`: comma-separated field names to ignore
- `--func-name`: custom function name for the **forward root** conversion only
- `--version`, `-v`: print version

## Development

Run tests:

```bash
go test ./...
```

Run benchmarks (including memory allocations):

```bash
go test -run '^$' -bench . -benchmem \
  ./internal/cli \
  ./internal/parser \
  ./internal/resolver \
  ./internal/generator
```

## CI

GitHub Actions workflows are included:

- CI: format check, `go vet`, `go test`
- Benchmark workflow: benchmark execution and artifact upload
- Release workflow: GoReleaser release on `v*` tags

## Dependency Updates

Renovate config is included (`renovate.json`) and configured to:

- keep Go module dependencies up to date
- keep GitHub Actions dependencies up to date
- keep the Go directive in `go.mod` updated

## Release Notes

Releases are handled by GoReleaser via `.goreleaser.yaml`.

For Homebrew publication, set this repository secret:

- `HOMEBREW_TAP_GITHUB_TOKEN`: PAT with push access to `seitarof/homebrew-tap`

## License

This project is licensed under the MIT License.

See [LICENSE](LICENSE).
