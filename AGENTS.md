# AGENTS.md — rawrequest

> Guidelines for AI agents operating in this repository.

## Project Overview

`rawrequest` is a Go CLI tool that sends raw HTTP requests via
[vodafon/rawhttp](https://github.com/vodafon/rawhttp). It reads a raw HTTP
request from stdin, optionally prefixed with a `#<URL>` target line, and prints
the response to stdout. Single-package (`main`), single-file project.

---

## Build / Run / Test Commands

### Build

```bash
go build -o rawrequest .
```

### Run (examples)

```bash
# With explicit target URL (first line starts with #)
echo -e '#https://example.com\nGET / HTTP/1.1\nHost: example.com\n\n' | go run . 

# With proxy
echo '...' | go run . -x http://127.0.0.1:8080

# Debug mode
echo '...' | go run . -debug
```

### Vet / Lint

```bash
go vet ./...
```

No `golangci-lint` config exists. If you add one, place it at `.golangci.yml`.

### Format

```bash
gofmt -w .
# or, if available:
gofumpt -w .
```

### Test

There are **no tests yet**. When adding tests:

```bash
# Run all tests
go test ./...

# Run a single test by name
go test -run TestFunctionName ./...

# Verbose
go test -v -run TestFunctionName ./...
```

Place test files alongside source as `main_test.go` (standard Go convention).

---

## Go Version & Dependencies

- **Go**: 1.25.5 (see `go.mod`)
- **Key dependency**: `github.com/vodafon/rawhttp` — raw HTTP client that
  preserves malformed/non-standard requests exactly as written.
- Indirect deps: `andybalholm/brotli`, `vodafon/vgutils`, `golang.org/x/net`

---

## Code Style Guidelines

### Formatting

- Use `gofmt` (or `gofumpt`). All code must pass `gofmt` with no diff.
- Tabs for indentation (Go standard).
- No trailing whitespace.

### Package & File Layout

- Single `main` package. Everything lives in `main.go` currently.
- If the project grows, split into focused files within the same package
  (e.g., `parse.go`, `client.go`). Do not create sub-packages unless there is a
  clear, reusable library boundary.

### Imports

- Group imports in standard Go style:
  1. Standard library
  2. (blank line)
  3. Third-party packages
- Use `goimports` ordering. No aliased imports unless there is a name collision.

```go
import (
	"bytes"
	"fmt"
	"io"

	"github.com/vodafon/rawhttp"
)
```

### Naming Conventions

- **Functions**: PascalCase for exported (`ParseData`), camelCase for unexported
  (`getHostRegex`, `dataNormalization`).
- **Variables**: camelCase. Package-level flags use `flag<Name>` prefix
  (`flagProxy`, `flagDebug`, `flagChangeCL`).
- **Types**: PascalCase structs (`Input`). Keep struct definitions near the top
  of the file, after `var` blocks.
- **Receivers**: not currently used — if added, use short 1-2 letter receivers
  matching the type initial (e.g., `func (i *Input) ...`).

### Error Handling

- Errors are returned as `error` values — never panic.
- In `main()`, errors are printed to **stderr** via `fmt.Fprintf(os.Stderr, ...)`
  and the process exits with `os.Exit(1)`.
- Non-main functions return errors to the caller; they do not print or exit.
- Error messages are lowercase, no trailing punctuation (Go convention):
  `fmt.Errorf("target line and host header missed, cant understand the target")`

### Types & Patterns

- No interfaces used — keep it simple for a CLI tool of this size.
- Prefer `[]byte` for request data manipulation (performance, avoids
  string ↔ []byte conversions).
- Use `bytes.*` functions for byte-slice operations rather than converting to
  string first.
- Compiled regexps (`regexp.MustCompile`) are created inside functions rather
  than as package-level vars — acceptable for this codebase's usage pattern, but
  if called in a loop, hoist to package level.

### CLI Flags

- Defined as package-level `var` block using `flag.String()` / `flag.Bool()`.
- Short flag names preferred (`-x`, `-cl`, `-debug`).
- `flag.Parse()` is the first call in `main()`.

### Output Conventions

- Normal output → `fmt.Printf` to **stdout**
- Error output → `fmt.Fprintf(os.Stderr, ...)` 
- Debug output → guarded by `*flagDebug`

### Comments

- Use `//` line comments. No block comments (`/* */`).
- Comments explain *why*, not *what* — especially for regex patterns and
  non-obvious byte manipulations.
- Functions that are non-trivial should have a brief comment, but this codebase
  is pragmatic — don't over-document obvious code.

---

## Architecture Notes

### Data Flow

```
stdin → ReadAll → ParseData → dataNormalization → rawhttp.Client.Do → stdout
```

### Input Formats

1. **With target line**: First line starts with `#`, followed by the URL.
   Request body follows on subsequent lines.
2. **Without target line**: Host header is extracted via regex to construct the
   URL. Scheme defaults to `https://`.

### Key Functions

| Function               | Purpose                                          |
|------------------------|--------------------------------------------------|
| `ParseData`            | Entry point for input parsing; dispatches format  |
| `ParseDataWithTarget`  | Handles `#<URL>\n<request>` format                |
| `dataNormalization`    | Normalizes line endings to `\r\n`                 |
| `replaceContentLength` | Replaces Content-Length with placeholder for rawhttp |
| `getHostRegex`         | Extracts Host header value via regex              |
| `replaceLastCR`        | Fixes trailing `\r` from editor artifacts         |

---

## Common Pitfalls

- **Line endings matter**: Raw HTTP requires `\r\n`. The `dataNormalization`
  function handles this — do not bypass it.
- **Content-Length**: By default (`-cl=true`), the tool replaces Content-Length
  with a placeholder that rawhttp recalculates. Do not hardcode Content-Length
  values in test fixtures when this flag is on.
- **stdin is blocking**: The tool reads ALL of stdin before proceeding. When
  testing interactively, send EOF (Ctrl+D) to trigger processing.

---

## Git Conventions

- Commit messages are short, lowercase, imperative
  (e.g., `fix nvim last CR issue`, `normalize requests`).
- No conventional commit prefixes (`feat:`, `fix:`) are used.
- `.gitignore` excludes the built binary (`rawrequest`) and `data/` directory.
