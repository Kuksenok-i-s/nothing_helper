.PHONY: help install-deps install-deps-debian run run-systray run-gio run-gio-lite run-gio-systray build build-systray build-gio build-gio-lite build-gio-systray build-helper build-gio-package prepare-debian debian-changelog package-deb package-arch package-arch-real package-rpm package-macos client-bundle-linux install-local vet test test-race check fmt lint clean profile-gio profile-gio-web sample-macos-app

.PHONY: run-lite build-lite build-package bundle-linux check-gui macos-app install-deps-arch

ARCH_BUILD_USER ?= builduser
GUI_TAGS ?= gio systray
BUNDLE_ARCH = $(shell go env GOARCH)
MACOS_PROCESS ?= Nothing_helper
.DEFAULT_GOAL := help

BINARY ?= nothing_helper
BINARY_GIO ?= $(BINARY)
BINARY_HELPER ?= nothing_helper_rfcomm_helper
CMD ?= ./cmd/nothing_helper
CMD_GIO ?= $(CMD)
CMD_HELPER ?= ./cmd/nothing_helper_rfcomm_helper
ARGS ?=

# X11/XWayland lets the desktop draw the system title bar on Linux.
.PHONY: run-native build-native
run-native:
	go run -tags "gio systray nowayland" $(CMD_GIO) $(ARGS)

build-native:
	go build -tags "gio systray nowayland" -o bin/$(BINARY_GIO) $(CMD_GIO)

help:
	@echo "Nothing_helper — build and development"
	@echo "  make run / build         GUI + tray (bin/$(BINARY_GIO))"
	@echo "  make run-native / build-native  System frame via X11/XWayland on Linux"
	@echo "  make run-lite / build-lite      GUI without tray; Gio libraries still required"
	@echo "  make build-helper        Linux RFCOMM helper (bin/$(BINARY_HELPER))"
	@echo "  make build-package       GUI + Linux helper"
	@echo "  make install-deps-debian Debian/Ubuntu build and runtime dependencies"
	@echo "  make install-deps-arch   Arch/Manjaro build and runtime dependencies"
	@echo "  make package-deb         Debian package in dist/ (PKG_VERSION=1.2.2)"
	@echo "  make package-arch        Arch package in dist/ (version from PKGBUILD)"
	@echo "  make package-rpm         Advanced: rpmbuild with a prepared source tree"
	@echo "  make bundle-linux        Linux binaries in dist/ (PKG_VERSION=1.2.2)"
	@echo "  make macos-app           Native Nothing_helper.app in dist/ (VERSION=1.2.2)"
	@echo "  make package-macos       Universal .app + DMG in dist/ (VERSION=1.2.2; macOS + CLT)"
	@echo "  make install-local       Linux install under /usr/local plus system polkit assets"
	@echo "  make vet / test / test-race / check  Core validation"
	@echo "  make check-gui           Vet + race tests with GUI_TAGS='$(GUI_TAGS)'"
	@echo "  make fmt / lint / clean  Format / vet / remove bin/"
	@echo "  make profile-gio / profile-gio-web  CPU profiling"
	@echo "  make sample-macos-app    Sample running $(MACOS_PROCESS) (MACOS_PROCESS override)"
	@echo ""
	@echo "Compatibility: run-gio, build-gio, *-systray, *-gio-lite, build-gio-package, client-bundle-linux."
	@echo "Flags: make run ARGS='--addr AA:BB:CC:DD:EE:FF --channel 15'"

install-deps: install-deps-debian

# Debian/Ubuntu packages for Bluetooth/RFCOMM, notifications, Gio, and systray.
install-deps-debian:
	sudo apt-get update
	sudo apt-get install -y \
		bluez \
		bluez-tools \
		build-essential \
		debhelper \
		dpkg-dev \
		libayatana-appindicator3-dev \
		libegl1-mesa-dev \
		libgl1-mesa-dev \
		libgles2-mesa-dev \
		libglib2.0-bin \
		libgtk-3-dev \
		libnotify-bin \
		libvulkan-dev \
		libwayland-dev \
		libx11-dev \
		libx11-xcb-dev \
		libxcursor-dev \
		libxfixes-dev \
		libxi-dev \
		libxinerama-dev \
		libxkbcommon-dev \
		libxkbcommon-x11-dev \
		libxrandr-dev \
		policykit-1 \
		pipewire-bin \
		pkg-config \
		rfkill

install-deps-arch:
	sudo pacman -S --needed base-devel go pkgconf gtk3 vulkan-headers libayatana-appindicator libxkbcommon-x11 bluez bluez-utils polkit libnotify pipewire

run-lite: run-gio-lite

build-lite: build-gio-lite

build-package: build-gio-package

bundle-linux: client-bundle-linux

run: run-gio

run-systray: run-gio

# Gio GUI with system tray (default). Requires libayatana-appindicator
# (Arch/Manjaro: pacman -S libayatana-appindicator) and a GNOME AppIndicator
# extension for the icon to appear on GNOME Shell.
run-gio:
	go run -tags "$(GUI_TAGS)" $(CMD_GIO) $(ARGS)

# Gio GUI without AppIndicator; the other Gio graphics libraries are still required.
run-gio-lite:
	go run -tags gio $(CMD_GIO) $(ARGS)

run-gio-systray: run-gio

build: build-gio

build-systray: build-gio

build-gio:
	go build -tags "$(GUI_TAGS)" -o bin/$(BINARY_GIO) $(CMD_GIO)

build-gio-lite:
	go build -tags gio -o bin/$(BINARY_GIO) $(CMD_GIO)

build-gio-systray: build-gio

build-helper:
	go build -o bin/$(BINARY_HELPER) $(CMD_HELPER)

build-gio-package: build-gio build-helper

prepare-debian:
	ln -sfn "$(CURDIR)/packaging/debian" "$(CURDIR)/debian"

debian-changelog:
	./scripts/set-debian-changelog.sh

package-deb: build-gio-package debian-changelog prepare-debian
	dpkg-buildpackage -us -uc -b -d
	@mkdir -p dist
	@cp ../nothing-helper_*.deb dist/ 2>/dev/null || true

client-bundle-linux: build-gio-package
	@mkdir -p dist
	@version="$${PKG_VERSION:-$$(./scripts/pkg-version.sh)}"; \
	arch="$(BUNDLE_ARCH)"; \
	if [ -n "$${ARCH:-}" ] && [ "$$ARCH" != "$$arch" ]; then echo "ARCH=$$ARCH does not match Go build architecture $$arch" >&2; exit 1; fi; \
	tar -czf "dist/Nothing_helper-$${version}-linux-$${arch}.tar.gz" \
		-C bin $(BINARY_GIO) $(BINARY_HELPER)

# makepkg refuses to run as root; CI Arch containers start as root.
package-arch:
	@if [ "$$(id -u)" -eq 0 ]; then \
		root_dir="$$(pwd)"; \
		getent passwd "$(ARCH_BUILD_USER)" >/dev/null 2>&1 || useradd -m -s /bin/bash "$(ARCH_BUILD_USER)"; \
		chown -R "$(ARCH_BUILD_USER):$(ARCH_BUILD_USER)" "$$root_dir"; \
		su "$(ARCH_BUILD_USER)" -c "cd \"$$root_dir\" && $(MAKE) package-arch-real"; \
	else \
		$(MAKE) package-arch-real; \
	fi

package-arch-real:
	cd packaging/arch && makepkg -sf --noconfirm
	@mkdir -p dist
	@cp packaging/arch/nothing_helper-*.pkg.tar.zst dist/ 2>/dev/null || true

package-rpm: build-gio-package
	rpmbuild -ba packaging/fedora/nothing_helper.spec

macos-app:
	./packaging/macos/bundle.sh

# Universal arm64+x86_64 .app bundle and DMG (run on macOS with Xcode CLT).
package-macos:
	chmod +x packaging/macos/*.sh
	./packaging/macos/package.sh

install-local: build-gio-package
	sudo install -Dm755 bin/$(BINARY) /usr/local/bin/$(BINARY)
	sudo install -Dm755 bin/$(BINARY_HELPER) /usr/local/libexec/$(BINARY_HELPER)
	sudo install -Dm644 packaging/common/nothing_helper.desktop /usr/local/share/applications/nothing_helper.desktop
	sudo sed -i "s|/usr/bin/nothing_helper|/usr/local/bin/$(BINARY)|g" /usr/local/share/applications/nothing_helper.desktop
	sudo install -Dm644 packaging/common/nothing_helper-autostart.desktop /etc/xdg/autostart/nothing_helper.desktop
	sudo sed -i "s|/usr/bin/nothing_helper|/usr/local/bin/$(BINARY)|g" /etc/xdg/autostart/nothing_helper.desktop
	sudo install -Dm644 packaging/common/nothing_helper.svg /usr/local/share/icons/hicolor/scalable/apps/nothing_helper.svg
	sudo install -Dm644 packaging/common/org.nothing_helper.rfcomm.policy /usr/share/polkit-1/actions/org.nothing_helper.rfcomm.policy
	sudo sed -i "s|/usr/libexec/nothing_helper_rfcomm_helper|/usr/local/libexec/$(BINARY_HELPER)|g" /usr/share/polkit-1/actions/org.nothing_helper.rfcomm.policy
	sudo install -Dm644 packaging/common/90-nothing_helper.rules /etc/polkit-1/rules.d/90-nothing_helper.rules
	sudo install -Dm644 packaging/common/nothing_helper.sysusers /usr/lib/sysusers.d/nothing_helper.conf

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

check: vet test test-race

check-gui:
	go vet -tags "$(GUI_TAGS)" ./...
	go test -race -tags "$(GUI_TAGS)" ./...

fmt:
	gofmt -w cmd internal

lint: vet

clean:
	rm -rf bin

PROFILE_SECONDS ?= 20
PROFILE_ADDR ?= 127.0.0.1:6060

profile-gio:
	chmod +x scripts/profile-gio.sh
	PROFILE_SECONDS=$(PROFILE_SECONDS) PROFILE_ADDR=$(PROFILE_ADDR) ./scripts/profile-gio.sh $(ARGS)

profile-gio-web:
	@prof=$$(ls -t captures/profiles/cpu-*.prof 2>/dev/null | head -1); \
	if [ -z "$$prof" ]; then echo "No profile in captures/profiles/; run make profile-gio first"; exit 1; fi; \
	go tool pprof -http=:8080 "$$prof"

sample-macos-app:
	chmod +x scripts/sample-macos-app.sh
	./scripts/sample-macos-app.sh "$(MACOS_PROCESS)" $(PROFILE_SECONDS)
