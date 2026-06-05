.PHONY: help install-deps install-deps-debian run run-systray run-gio run-gio-lite run-gio-systray build build-systray build-gio build-gio-lite build-gio-systray build-helper build-gio-package package-deb package-arch package-rpm install-local test fmt lint clean

BINARY ?= tws_manager
BINARY_GIO ?= tws_manager_gio
BINARY_HELPER ?= tws_manager_rfcomm_helper
CMD ?= ./cmd/tws_manager
CMD_GIO ?= ./cmd/tws_manager_gio
CMD_HELPER ?= ./cmd/tws_manager_rfcomm_helper
ARGS ?=

help:
	@echo "tws_manager targets:"
	@echo "  make install-deps        Install Debian/Ubuntu system packages"
	@echo "  make run                 Run TUI CLI"
	@echo "  make run-systray         Run TUI CLI with systray support"
	@echo "  make run-gio             Run Gio GUI with systray support"
	@echo "  make run-gio-lite        Run Gio GUI without systray"
	@echo "  make build               Build TUI CLI to bin/$(BINARY)"
	@echo "  make build-systray       Build TUI CLI with systray support"
	@echo "  make build-gio           Build Gio GUI with systray support"
	@echo "  make build-gio-lite      Build Gio GUI without systray"
	@echo "  make build-helper        Build root helper binary"
	@echo "  make build-gio-package   Build Gio + helper binaries"
	@echo "  make package-deb         Build Debian package (dpkg-buildpackage)"
	@echo "  make package-arch        Prepare Arch package (PKGBUILD)"
	@echo "  make package-rpm         Build Fedora RPM (rpmbuild)"
	@echo "  make install-local       Install assets under /usr/local (needs sudo)"
	@echo "  make test                Run all tests"
	@echo "  make fmt                 Format Go code"
	@echo "  make lint                Run lint checks"
	@echo "  make clean               Remove build outputs"
	@echo ""
	@echo "Use ARGS='...' to pass flags, for example:"
	@echo "  make run ARGS='--device /dev/rfcomm0'"

install-deps: install-deps-debian

# Debian/Ubuntu packages for Bluetooth/RFCOMM, notifications, Gio, and systray.
install-deps-debian:
	sudo apt-get update
	sudo apt-get install -y \
		bluez \
		bluez-tools \
		build-essential \
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
		libxcursor-dev \
		libxi-dev \
		libxinerama-dev \
		libxkbcommon-dev \
		libxkbcommon-x11-dev \
		libxrandr-dev \
		pkg-config \
		rfkill

run:
	go run $(CMD) $(ARGS)

run-systray:
	go run -tags systray $(CMD) $(ARGS)

# Gio GUI with system tray (default). Requires libayatana-appindicator
# (Arch/Manjaro: pacman -S libayatana-appindicator) and a GNOME AppIndicator
# extension for the icon to appear on GNOME Shell.
run-gio:
	go run -tags "gio systray" $(CMD_GIO) $(ARGS)

# Gio GUI without the tray (no extra system libraries required).
run-gio-lite:
	go run -tags gio $(CMD_GIO) $(ARGS)

run-gio-systray: run-gio

build:
	go build -o bin/$(BINARY) $(CMD)

build-systray:
	go build -tags systray -o bin/$(BINARY) $(CMD)

build-gio:
	go build -tags "gio systray" -o bin/$(BINARY_GIO) $(CMD_GIO)

build-gio-lite:
	go build -tags gio -o bin/$(BINARY_GIO) $(CMD_GIO)

build-gio-systray: build-gio

build-helper:
	go build -o bin/$(BINARY_HELPER) $(CMD_HELPER)

build-gio-package: build-gio build-helper build

package-deb: build-gio-package
	dpkg-buildpackage -us -uc -b -d

package-arch: build-gio-package
	@echo "Use packaging/arch/PKGBUILD with makepkg in an Arch build environment."

package-rpm: build-gio-package
	rpmbuild -ba packaging/fedora/tws_manager.spec

install-local: build-gio-package
	sudo install -Dm755 bin/$(BINARY) /usr/local/bin/$(BINARY)
	sudo install -Dm755 bin/$(BINARY_GIO) /usr/local/bin/$(BINARY_GIO)
	sudo install -Dm755 bin/$(BINARY_HELPER) /usr/local/libexec/$(BINARY_HELPER)
	sudo install -Dm644 packaging/common/tws_manager.desktop /usr/local/share/applications/tws_manager.desktop
	sudo install -Dm644 packaging/common/tws_manager-autostart.desktop /etc/xdg/autostart/tws_manager.desktop
	sudo install -Dm644 packaging/common/tws_manager.svg /usr/local/share/icons/hicolor/scalable/apps/tws_manager.svg
	sudo install -Dm644 packaging/common/org.tws_manager.rfcomm.policy /usr/share/polkit-1/actions/org.tws_manager.rfcomm.policy
	sudo install -Dm644 packaging/common/90-tws_manager.rules /etc/polkit-1/rules.d/90-tws_manager.rules
	sudo install -Dm644 packaging/common/tws_manager.sysusers /usr/lib/sysusers.d/tws_manager.conf

test:
	go test ./...

fmt:
	gofmt -w cmd internal

lint:
	go test ./...

clean:
	rm -rf bin
