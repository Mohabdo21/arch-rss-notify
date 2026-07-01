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
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release VERSION=x.y.z"; \
		exit 1; \
	fi; \
	CURRENT_VER=$$(grep 'const version' main.go | cut -d'"' -f2); \
	echo ""; \
	echo "$(BINARY) v$$CURRENT_VER -> v$(VERSION)"; \
	read -p "Proceed? [y/N] " ok; \
	if [ "$$ok" != "y" ]; then exit 1; fi; \
	echo ""; \
	echo "==> Bumping version in main.go, aur/PKGBUILD, and aur/.SRCINFO"; \
	sed -i "s/const version = \"$$CURRENT_VER\"/const version = \"$(VERSION)\"/" main.go; \
	sed -i "s/^pkgver=.*/pkgver=$(VERSION)/" aur/PKGBUILD; \
	sed -i "s/^pkgrel=.*/pkgrel=1/" aur/PKGBUILD; \
	sed -i "s/^pkgver =.*/pkgver = $(VERSION)/" aur/.SRCINFO; \
	sed -i "s/^pkgrel =.*/pkgrel = 1/" aur/.SRCINFO; \
	sed -i "s|source =.*|source = arch-rss-notify-$(VERSION).tar.gz::https://github.com/Mohabdo21/arch-rss-notify/archive/v$(VERSION).tar.gz|" aur/.SRCINFO; \
	git add main.go aur/PKGBUILD aur/.SRCINFO; \
	git commit -m "chore: bump version to $(VERSION)"; \
	echo ""; \
	echo "==> Tagging v$(VERSION)"; \
	git tag "v$(VERSION)"; \
	echo ""; \
	echo "==> Pushing to GitHub"; \
	git push origin main "v$(VERSION)"; \
	echo ""; \
	echo "==> Computing SHA256 for source tarball"; \
	TAR_URL="https://github.com/Mohabdo21/arch-rss-notify/archive/v$(VERSION).tar.gz"; \
	SHA=$$(curl -sL "$$TAR_URL" | sha256sum | cut -d' ' -f1); \
	sed -i "s/^sha256sums=.*/sha256sums=('$$SHA')/" aur/PKGBUILD; \
	sed -i "s/^sha256sums =.*/sha256sums = $$SHA/" aur/.SRCINFO; \
	git add aur/PKGBUILD aur/.SRCINFO; \
	git commit -m "chore: update AUR PKGBUILD checksums for v$(VERSION)"; \
	git push origin main; \
	echo ""; \
	echo "==> Publishing to AUR"; \
	$(MAKE) aur-publish; \
	echo ""; \
	echo "Release v$(VERSION) complete."

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
	@cp aur/PKGBUILD $(AUR_DIR)/PKGBUILD
	@cp aur/arch-rss-notify.install $(AUR_DIR)/arch-rss-notify.install 2>/dev/null || true
	@CURRENT_VER=$$(grep '^pkgver=' $(AUR_DIR)/PKGBUILD | cut -d= -f2); \
	CURRENT_REL=$$(grep '^pkgrel=' $(AUR_DIR)/PKGBUILD | cut -d= -f2); \
	NEW_VER=$(VERSION); \
	if [ "$$CURRENT_VER" != "$$NEW_VER" ]; then \
		echo "Version changed: $$CURRENT_VER -> $$NEW_VER"; \
		sed -i "s/^pkgver=.*/pkgver=$$NEW_VER/" $(AUR_DIR)/PKGBUILD; \
		sed -i "s/^pkgrel=.*/pkgrel=1/" $(AUR_DIR)/PKGBUILD; \
		echo "pkgrel reset to 1"; \
	else \
		echo "Version unchanged: $$CURRENT_VER"; \
		read -p "Increment pkgrel? (y/n): " inc; \
		if [ "$$inc" = "y" ]; then \
			NEW_REL=$$((CURRENT_REL + 1)); \
			sed -i "s/^pkgrel=.*/pkgrel=$$NEW_REL/" $(AUR_DIR)/PKGBUILD; \
			echo "pkgrel incremented to $$NEW_REL"; \
		else \
			echo "pkgrel left as $$CURRENT_REL"; \
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
