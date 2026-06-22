# tuidledo

[![Go](https://github.com/sgruendel/tuidledo/actions/workflows/go.yml/badge.svg)](https://github.com/sgruendel/tuidledo/actions/workflows/go.yml)

Go TUI client for Toodledo, focused on a Master Your Now style task list.

## Status

Early scaffold. The first version supports:

- Toodledo API v3 OAuth login with a localhost callback
- MYN-style task filtering
- Toodledo contexts for separate work/private task lists
- Vim/lazygit-style navigation
- Task creation with MYN defaults
- Task completion

## Setup

Register an app with Toodledo and configure this redirect URI:

```text
http://127.0.0.1:8765/callback
```

Export your app credentials:

```sh
export TOODLEDO_CLIENT_ID="your-client-id"
export TOODLEDO_CLIENT_SECRET="your-client-secret"
```

Run the TUI:

```sh
go run ./cmd/tuidledo
```

On first launch, open the printed authorization URL and approve access. Tokens and the last selected context are stored in your user config directory as `tuidledo/config.json`.

## MYN Filtering

The task list hides:

- completed tasks
- negative-priority tasks
- tasks with a future start date

Visible tasks are sorted by priority first, then start date descending within each priority group.

New tasks use the current context, medium priority, and today's start date.

## Keybindings

- `j` / `k` or arrows: move selection
- `g` / `G`: jump to top/bottom
- `tab` / `shift+tab`: jump between priority groups
- `[` / `]`: switch context
- `/`: search visible task titles
- `n`: create task in the current context
- `space`: complete selected task
- `enter`: show task details
- `r`: refresh from Toodledo
- `esc`: back or clear search
- `?`: help
- `q`: back, or quit from the task list
- `ctrl+c`: quit
