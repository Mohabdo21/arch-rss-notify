# arch-rss-notify - AGENTS.md

## Project

Go 1.26.4 single-main-package tool (`github.com/Mohabdo21/arch-rss-notify`) that polls Arch Linux package RSS feeds and sends `notify-send` desktop notifications when installed packages have updates.

## Commands

- `make build` - build to `bin/rss_notify` (CGO_ENABLED=0)
- `make build-static` - fully static amd64, stripped, trimpath
- `make test` - runs `make check` first, then `go test -race -v ./...`
- `make check` - runs `go fmt`, `go fix`, `go vet`, `golangci-lint fmt`, `golangci-lint run --fix`

Order: `make check` then `make test`.

## Key facts

- Single `package main` - no internal packages or cmd/ layout
- Uses Go 1.26 features: `strings.SplitSeq`, `errors.AsType`
- Config from `.env` file (via godotenv) + CLI flags `--interval`, `--state`
- Default state file: `~/.local/share/rss-notifier/state.json`
- State file only writes to disk when dirty (optimization)
- `golangci-lint` enforces 80-char line limit
- pre-commit hooks run `make check` (auto-formats + lints + vets)
- Works on Arch Linux only (calls `pacman -Q` and `pacman -Qi`)
- Requires `notify-send` (libnotify) and D-Bus session bus at runtime

## Testing quirks

- `pacman_test.go` calls real `pacman -Q` - only meaningful on Arch Linux
- `config_test.go` depends on env vars (no env cleanup) - run with clean env or accept test depends on `.env`
- State tests use a tmp file, cleanup with `os.Remove`

## AUR release

```sh
make release VERSION=x.y.z             # bump, tag, push, update checksums, publish to AUR
make aur-clone                         # clone AUR repo to /tmp/rss-notify-aur
make aur-update                        # bump pkgver, reset pkgrel, update checksums
make aur-publish                       # commit + push to AUR if changed
```

- `rss-notify.service` is shipped at `/usr/lib/systemd/user/`
- `.SRCINFO` is updated by `make release` (sha256sums synced with PKGBUILD)
