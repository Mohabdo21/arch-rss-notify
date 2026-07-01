# arch-rss-notify

Polls Arch Linux package RSS feeds and sends desktop notifications when installed packages have updates available.

## Features

- Checks core/extra/multilib Arch RSS feeds for new package versions
- Cross-references against all installed packages (`pacman -Q`)
- Deduplicates notifications by tracking last-notified version in a JSON state file
- Critical package detection (linux, nvidia, mesa, glibc, systemd, etc.) sends urgency=critical notifications
- Configurable check interval (default: 10m, flag: `--interval`)
- Configurable state file path (flag: `--state`)
- Concurrent RSS fetching with exponential backoff + jitter on transient errors
- Graceful shutdown via SIGINT/SIGTERM (saves state before exit)

## Requirements

- Arch Linux
- Go 1.26+ for building
- `notify-send` (libnotify) for desktop notifications
- D-Bus session bus (desktop environment)

## Installation

```sh
git clone https://github.com/Mohabdo21/arch-rss-notify.git
cd arch-rss-notify

make build          # build to bin/rss_notify (CGO_ENABLED=0)
make build-static   # fully static amd64 binary
make test           # run tests

# or just:
go build -o rss_notify .
```

## Configuration

Settings are loaded from a `.env` file in the working directory (if present), then overridden by CLI flags. An example `.env` is provided in the repository -- copy and edit it to your needs.

| Variable         | Default                                       | Description           |
| ---------------- | --------------------------------------------- | --------------------- |
| `RSS_URL`        | core, extra, multilib feeds (comma-separated) | RSS feed URLs         |
| `CHECK_INTERVAL` | `10m`                                         | Poll interval         |
| `STATE_FILE`     | `~/.local/share/rss-notifier/state.json`      | State file path       |
| `FETCH_RETRIES`  | `2`                                           | Max retries per feed  |
| `FETCH_BACKOFF`  | `1s`                                          | Initial retry backoff |

CLI flags override corresponding env vars:

```
--interval   check interval (e.g. "10m", "30s")
--state      path to state JSON file
```

## Usage

```sh
./rss_notify
```

Runs as a foreground process. Checks feeds every 10 minutes (or configured interval). Sends `notify-send` notifications when updates are found. Press Ctrl+C to stop.

### systemd user service

```ini
[Unit]
Description=Arch RSS Package Update Notifier
After=network.target

[Service]
Type=simple
ExecStart=/path/to/rss_notify
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
```

Place at `~/.config/systemd/user/rss-notifier.service`, then enable:

```sh
systemctl --user daemon-reload
systemctl --user enable --now rss-notifier.service
```
