# tuidledo

[![build](https://github.com/sgruendel/tuidledo/actions/workflows/go.yml/badge.svg)](https://github.com/sgruendel/tuidledo/actions/workflows/go.yml)

Go TUI client for Toodledo, focused on a Master Your Now style task list.

## Status

Version 0.1 is focused on core MYN task management rather than full Toodledo
feature coverage. It supports:

- Toodledo API v3 OAuth login with a localhost callback
- MYN-style task filtering
- Toodledo contexts for separate work/private task lists
- Vim/lazygit-style navigation
- Task creation with MYN defaults
- Task completion
- Task deletion with confirmation
- Task details and editing for title, note, priority, start date, due date, and context

Not currently supported: folders, goals, tags, subtasks, saved searches, full
repeat editing, file attachment management, or non-MYN list modes.

## Setup

Prebuilt release binaries can embed the shared Toodledo OAuth client credentials.
Locally, `config.json` stores only access tokens, refresh tokens, token expiry, and
the last selected context.

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

If you are running an official release binary, you do not need to set
`TOODLEDO_CLIENT_ID` or `TOODLEDO_CLIENT_SECRET` because those values are embedded
at build time by GitHub Actions.

Print the version:

```sh
tuidledo --version
```

Release builds can stamp the version with:

```sh
go build -ldflags "-X main.version=0.1.0" ./cmd/tuidledo
```

To build a distributable binary with embedded Toodledo credentials yourself:

```sh
go build -ldflags "-X main.version=0.1.0 -X main.clientID=$TOODLEDO_CLIENT_ID -X main.clientSecret=$TOODLEDO_CLIENT_SECRET" ./cmd/tuidledo
```

## Releases

The GitHub Actions workflow does three things:

- runs formatting, tests, and a normal build on pushes and pull requests
- builds downloadable archives for Linux x86_64, Windows x86_64, macOS x86_64, and macOS arm64 on tags like `v0.1.0`
- publishes a matching `.sha256` checksum file for each release archive

To enable release builds, add these repository secrets:

- `TOODLEDO_CLIENT_ID`
- `TOODLEDO_CLIENT_SECRET`

Then create and push a version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

That tag run uploads workflow artifacts and publishes the packaged binaries plus
their checksum files as GitHub release assets.

On first launch, open the printed authorization URL and approve access. Tokens
and the last selected context are stored in your user config directory as
`tuidledo/config.json`.

## MYN Filtering

The task list hides:

- completed tasks
- negative-priority tasks
- tasks with a future start date

Visible tasks are sorted by priority first, then start date descending within
each priority group.

New tasks use the current context, medium priority, and today's start date.

## Keybindings

- `j` / `k` or arrows: move selection
- `g` / `G`: jump to top/bottom
- `.` / `,`: jump to next/previous priority group
- `tab` / `shift+tab`: same as `.` / `,`
- `h` / `l`: collapse/expand active priority group
- `enter`: expand/collapse priority header or open selected task details
- `[` / `]`: switch context
- `/`: search visible task titles
- `n`: create new task in the current context
- `d`: mark selected task done
- `D`: ask to delete selected task
- `e`: edit task from task details
- `enter`: show task details
- `r`: refresh from Toodledo
- `esc`: back or clear search
- `?`: help
- `q`: back, or quit from the task list
- `ctrl+c`: quit

In the edit form, `tab` / `shift+tab` switches between fields, `ctrl+s` saves, and `esc` cancels. Priority and context fields cycle with `[` / `]` or `enter`. Date fields use `h` / `j` / `k` / `l`, `H` / `L` for months, `enter` to select, and `x` to clear.

## tmux Hyperlinks

Task notes use terminal hyperlinks for URLs. If links work directly in your
terminal but not inside tmux, enable passthrough in `~/.tmux.conf`:

```tmux
set -g allow-passthrough on
set -as terminal-features ',*:hyperlinks'
```

Restart tmux or reload the config after changing it.
