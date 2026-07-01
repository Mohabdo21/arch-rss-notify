# arch-rss-notify

Desktop notifications when Arch Linux packages have updates.

Monitors the core, extra, and multilib RSS feeds and notifies you when an installed package has a newer version.

## Requirements

- Arch Linux
- `notify-send` (libnotify)
- D-Bus session bus (desktop environment)
- Go 1.26+ (source builds only - not needed for AUR install)

## Installation

### From AUR (recommended)

```sh
yay -S arch-rss-notify
# or
paru -S arch-rss-notify
```

Or manually:

```sh
git clone https://aur.archlinux.org/arch-rss-notify.git
cd arch-rss-notify
makepkg -si
```

### systemd user service

For AUR installs, the post-install message shows the enable command.
Place `.env` at `~/.config/rss-notify/.env` first (see Configuration).

For source builds, copy the service file and enable manually:

```sh
cp rss-notify.service ~/.config/systemd/user/
systemctl --user enable --now rss-notify.service
```

### Build from source

```sh
git clone https://github.com/Mohabdo21/arch-rss-notify.git
cd arch-rss-notify

make build          # bin/rss_notify
make build-static   # fully static amd64 binary
make test           # run tests
```

## Configuration

Settings are loaded from `.env` (working directory) and overridden by CLI flags.

| Install method | `.env` location                                          |
| -------------- | -------------------------------------------------------- |
| AUR (systemd)  | `~/.config/rss-notify/.env`                              |
| AUR (manual)   | working directory, or use `--interval` / `--state` flags |
| Local build    | next to the binary, or project root                      |

An example `.env` is provided in the repository.

| Variable         | Default                                       | Description      |
| ---------------- | --------------------------------------------- | ---------------- |
| `RSS_URL`        | core, extra, multilib feeds (comma-separated) | Feed URLs        |
| `CHECK_INTERVAL` | `10m`                                         | Poll interval    |
| `STATE_FILE`     | `~/.local/share/rss-notifier/state.json`      | State file path  |
| `FETCH_RETRIES`  | `2`                                           | Max retries/feed |
| `FETCH_BACKOFF`  | `1s`                                          | Initial backoff  |

CLI flags:

```
--interval   check interval (e.g. "10m", "30s")
--state      path to state JSON file
```

## Usage

```sh
# AUR install
rss-notify

# Local build
./rss_notify
```

Runs until Ctrl+C. Checks feeds every 10 minutes (or configured interval) and sends `notify-send` notifications for updates.
