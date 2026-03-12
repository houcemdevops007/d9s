# d9s

> A modern TUI for Docker & Docker Compose, keyboard-first, inspired by k9s.

![d9s TUI](https://img.shields.io/badge/d9s-v0.1.0-brightgreen)
![Go](https://img.shields.io/badge/Go-1.21+-blue)
![License](https://img.shields.io/badge/License-MIT-yellow)

## Features (V1)

- **Docker Contexts** — list, view current, switch between contexts
- **Compose Projects** — discover all projects via `docker compose ls`
- **Containers** — list all (running + stopped), with state indicators
- **Logs** — tail container logs in real-time
- **Events** — live Docker daemon event stream
- **Stats** — CPU%, memory usage per container
- **Inspect** — full JSON inspect panel
- **Actions** — start, stop, restart, remove, compose up/down/pull/build, exec shell
- **Search** — fuzzy filter containers with `/`
- **Keyboard-first UX** — no mouse required

## Requirements

- macOS or Linux
- Go 1.21+
- Docker Engine running (local socket `/var/run/docker.sock`)
- `docker compose` CLI available in PATH

## Installation

```bash
# Build from source
make build

# Or run directly
go run ./cmd/d9s
```

## Key Bindings

| Key         | Action                            |
|-------------|-----------------------------------|
| `Tab`       | Switch panel (Contexts→Projects→Containers) |
| `↑` / `↓`  | Navigate                          |
| `/`         | Search containers                 |
| `l`         | View Logs                         |
| `e`         | View Events                       |
| `i`         | Inspect container                 |
| `s`         | Stats view                        |
| `S`         | Open shell (exec)                 |
| `r`         | Restart container                 |
| `x`         | Stop container                    |
| `R` / `Del` | Remove container                  |
| `u`         | Compose up                        |
| `d`         | Compose down                      |
| `p`         | Compose pull                      |
| `b`         | Compose build                     |
| `?`         | Toggle help                       |
| `q` / `^C`  | Quit                              |

## Layout

```
┌─────────────────────────────────────────────────────────────┐
│ ⬡ d9s                                              v0.1.0   │
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
│ CONTEXTS      │ CONTAINERS              │ [Logs] Events Stats│
│ ✓ default     │ ID   NAME   IMAGE STATE│                    │
│               │─────────────────────── │                    │
│ PROJECTS      │ abc1 nginx  nginx:latest│                    │
│ ● myapp       │ def2 api    myapp:dev  │                    │
│ ⬜ infra      │                        │                    │
─────────────────────────────────────────────────────────────
│ Tab panel  ↑↓ nav  l logs  e events  s shell  r restart  q quit │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

Config file: `~/.config/d9s/config.json`

```json
{
  "default_context": "",
  "stats_interval": "2s",
  "log_tail_lines": 200,
  "theme": "dark",
  "refresh_interval": "5s"
}
```

## Architecture

```
d9s/
  cmd/d9s/        — entry point
  internal/
    app/          — main event loop, wiring
    domain/       — business models (Container, ComposeProject, ...)
    dockerapi/    — Docker REST API client (Unix socket, stdlib only)
    compose/      — docker compose CLI wrapper
    store/        — central state (RWMutex, pub/sub, selectors)
    tui/          — ANSI TUI (view, style, terminal I/O)
    actions/      — user-facing actions (restart, exec, etc.)
    config/       — JSON config
  pkg/version/    — version info
```

## Development

```bash
make build       # build binary
make run         # run locally
make test        # run unit tests
make fmt         # gofmt
make lint        # go vet
make clean       # clean artifacts
```

## License

MIT — see [LICENSE](LICENSE)
