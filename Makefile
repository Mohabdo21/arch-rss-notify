SHELL := /bin/bash

BINARY = rss_notify
BUILD_DIR = bin
AUR_REPO = ssh://aur@aur.archlinux.org/arch-rss-notify.git
AUR_DIR = /tmp/arch-rss-notify-aur
VERSION = $(shell grep 'const version' main.go | cut -d'"' -f2)

.PHONY: build build-static test check release aur-clone aur-update aur-publish

build:
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) .

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-trimpath \
		-mod=readonly \
		-o $(BUILD_DIR)/$(BINARY)_static .

test: check
	@echo "Running tests..."
	go test -race -v ./...

check:
	go fmt ./...
	go fix ./...
	go vet ./...
	golangci-lint fmt
	golangci-lint run --fix

# --- Release ---

release:
	@scripts/release.sh "$(VERSION)"

# --- AUR ---

aur-clone:
	@if [ ! -d "$(AUR_DIR)" ]; then \
		git clone $(AUR_REPO) $(AUR_DIR); \
	else \
		echo "AUR repo already cloned at $(AUR_DIR)"; \
	fi

aur-update: aur-clone
	@cd $(AUR_DIR) && \
		if git rev-parse --verify HEAD >/dev/null 2>&1; then \
			git pull; \
		else \
			echo "Empty AUR repo -- skipping pull"; \
			git branch -m master; \
		fi
	@AUR_PKGVER=$$(grep '^pkgver=' $(AUR_DIR)/PKGBUILD | cut -d= -f2); \
	AUR_PKGREL=$$(grep '^pkgrel=' $(AUR_DIR)/PKGBUILD | cut -d= -f2); \
	cp aur/PKGBUILD $(AUR_DIR)/PKGBUILD; \
	cp aur/arch-rss-notify.install $(AUR_DIR)/arch-rss-notify.install 2>/dev/null || true; \
	NEW_VER=$(VERSION); \
	if [ "$$AUR_PKGVER" != "$$NEW_VER" ]; then \
		echo "Version changed: $$AUR_PKGVER -> $$NEW_VER (pkgrel reset to 1)"; \
		sed -i "s/^pkgver=.*/pkgver=$$NEW_VER/" $(AUR_DIR)/PKGBUILD; \
		sed -i "s/^pkgrel=.*/pkgrel=1/" $(AUR_DIR)/PKGBUILD; \
	else \
		echo "Version unchanged (published AUR is already at $$AUR_PKGVER-$$AUR_PKGREL)"; \
		read -p "Increment pkgrel? (y/n): " inc; \
		if [ "$$inc" = "y" ]; then \
			NEW_REL=$$((AUR_PKGREL + 1)); \
			sed -i "s/^pkgrel=.*/pkgrel=$$NEW_REL/" $(AUR_DIR)/PKGBUILD; \
			echo "pkgrel incremented to $$NEW_REL"; \
		else \
			echo "pkgrel left as $$AUR_PKGREL"; \
		fi \
	fi
	@echo "Computing SHA256 for source tarball..."
	@cd $(AUR_DIR) && SHA=$$(makepkg -g 2>/dev/null | grep -oP "'\K[^']+" | head -1) && \
		sed -i "s/^sha256sums=.*/sha256sums=('$$SHA')/" PKGBUILD
	@cd $(AUR_DIR) && makepkg --printsrcinfo > .SRCINFO
	@echo "PKGBUILD and .SRCINFO updated for $(VERSION)"

aur-publish: aur-update
	@cd $(AUR_DIR) && \
		if [ -n "$$(git status --porcelain PKGBUILD .SRCINFO)" ]; then \
			git add PKGBUILD .SRCINFO arch-rss-notify.install && \
			git commit -m "Update to $(VERSION)" && \
			git push origin master && \
			echo "Published arch-rss-notify to AUR"; \
		else \
			echo "No changes to commit."; \
		fi
