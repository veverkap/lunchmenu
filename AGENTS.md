# AGENTS.md

## Project Overview

This repository contains a scheduled Go application that prepares the next school day's lunch update. It fetches menus for two schools, adds a weather forecast, optionally enhances the message with GitHub Models, adds an API League riddle and GIF, writes dated artifacts, and posts the result to Telegram.

The application is designed primarily for unattended execution from GitHub Actions. Treat local end-to-end runs as side-effecting: they call live services, write files under `menus/` and `riddles/`, and may send a real Telegram message.

## Repository Structure

- `cmd/cli/main.go`: composition root and end-to-end workflow.
- `internal/lunch/config.go`: required environment variables, school configuration, prompt text, and New York date handling.
- `internal/lunch/menu.go`: SchoolCafe menu retrieval and parsing.
- `internal/lunch/weather.go`: wttr.in forecast retrieval and formatting.
- `internal/lunch/ai.go`: GitHub Models client and AI enhancement.
- `internal/lunch/api-league.go`: API League GIF and riddle clients.
- `internal/lunch/telegram.go`: Telegram Bot API client.
- `menus/`: generated plain-text lunch messages, named `YYYY-MM-DD.txt`.
- `riddles/`: generated riddle responses, named `YYYY-MM-DD.txt`.
- `index.html`, `imgs/`, and `page_hash.txt`: static GitHub Pages content and supporting assets.
- `.github/workflows/go_runner.yml`: scheduled production workflow.
- `lunch.prompt.yml`: reference prompt content; the runtime system message currently lives in `internal/lunch/config.go`.

## Tech Stack

- Go version and dependencies are defined by `go.mod`; do not duplicate or guess versions elsewhere.
- `github.com/buger/jsonparser` is used for external JSON traversal.
- `github.com/openai/openai-go/v3` calls GitHub Models.
- `github.com/hashicorp/go-retryablehttp` is used for Telegram delivery.
- External services are SchoolCafe, wttr.in, GitHub Models, API League, and Telegram Bot API.

## Build and Run

```bash
go mod download
make build
```

`make build` produces the ignored/generated `lunch_menu` binary from `./cmd/cli`. The equivalent direct command is:

```bash
go build -o lunch_menu ./cmd/cli
```

Running the binary requires all of these environment variables:

- `TELEGRAM_TOKEN`
- `TELEGRAM_CHESAPEAKE_CHAT_ID`
- `GH_TOKEN`
- `API_LEAGUE_KEY`

Only run `./lunch_menu` when live API calls, generated files, and a Telegram send are intended. Never print, log, commit, or place real secret values in fixtures.

## Testing and Validation

There are currently no dedicated test files. For every Go change, run the narrowest relevant checks, followed by the repository-wide checks before handing off:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
make build
```

`go test ./...` is still useful as a package compilation check. Prefer `httptest.Server`, injected clients, and deterministic dates when adding tests; tests must not call production services or send Telegram messages.

## Patterns and Conventions

- Keep `cmd/cli/main.go` focused on orchestration. Put service-specific behavior in `internal/lunch`.
- Follow existing Go formatting and naming, and keep exported identifiers documented when introducing public package APIs.
- Use `log/slog` structured logging with message text plus key/value context. Return errors to the orchestration layer unless the operation is explicitly best-effort.
- Give outbound HTTP clients an explicit timeout, check for `http.StatusOK`, read and close response bodies, and wrap errors with operation context.
- Continue using `jsonparser` for external API responses unless a change clearly benefits from a stable typed response model.
- Preserve fallback behavior: weather, AI enhancement, riddles, and GIFs are optional additions; failure of the core menu retrieval should stop the run.
- Keep Telegram output compatible with its legacy `Markdown` parse mode. Escape or constrain newly introduced dynamic formatting where necessary.
- Keep dates in `America/New_York` and normalize date-only values to noon to avoid DST boundary errors. Generated filenames use `YYYY-MM-DD`; user-facing dates use `MM/DD/YYYY`.
- Do not edit dated files under `menus/` or `riddles/` as part of unrelated code changes. They are production output committed by automation.

## Changing Application Behavior

When adding or changing an integration:

1. Put credentials and static configuration in `internal/lunch/config.go`, validate required values in `NewConfig`, and add the corresponding GitHub Actions secret mapping.
2. Implement service behavior in a focused file under `internal/lunch`, using a client with a timeout and explicit error handling.
3. Wire the client into `cmd/cli/main.go`, preserving which failures are fatal versus best-effort.
4. Add deterministic tests around parsing, formatting, and fallback behavior without using live endpoints.
5. Update this file and `.github/copilot-instructions.md` when paths, commands, environment variables, or architecture change.

If the AI prompt changes, update the runtime `SystemMessage` in `internal/lunch/config.go`. Keep `lunch.prompt.yml` synchronized if it remains a maintained reference.

## CI/CD

`.github/workflows/go_runner.yml` runs daily at `14:05 UTC` and can also be started manually. It installs Go, builds the CLI, runs it with repository secrets, then commits generated changes to `main`.

Changes to the Go version must keep `go.mod` and the workflow's `actions/setup-go` configuration compatible. Changes to output paths must also update the workflow's commit assumptions and any GitHub Pages links built in `cmd/cli/main.go`.

## Common Pitfalls

- A local end-to-end run can send a real message and mutate tracked generated content.
- `NewConfig` fails fast if any required environment variable is absent.
- The orchestration assumes exactly two configured schools and at least one combined menu item; preserve or explicitly revise those invariants when changing school handling.
- Do not treat `lunch.prompt.yml` as the runtime prompt without also checking `internal/lunch/config.go`.
- Avoid `go mod tidy` as an install-only CI step when no dependency change is intended because it can modify `go.mod` or `go.sum`.