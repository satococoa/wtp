# Dev Commands

Use Task commands as the default interface for local development.

- Build: `go tool task build`
- Install: `go tool task install`
- Test: `go tool task test`
- Lint: `go tool task lint`
- Format: `go tool task fmt`
- E2E: `go tool task test-e2e`
- Full local check: `go tool task dev`

Fast feedback loop:
- `go run ./cmd/wtp <args>`

Notes:
- `go tool task test` writes `coverage.out`.
- `go tool task test-e2e` uses the built binary and supports `WTP_E2E_BINARY=/abs/path/wtp`.
