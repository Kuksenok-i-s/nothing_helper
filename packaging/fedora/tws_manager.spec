Name:           tws_manager
Version:        0.1.0
Release:        1%{?dist}
Summary:        TWS RFCOMM desktop client
License:        MIT
URL:            https://github.com/example/tws_manager
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang
BuildRequires:  pkgconfig
BuildRequires:  gtk3-devel
BuildRequires:  libayatana-appindicator-gtk3-devel
BuildRequires:  vulkan-headers
Requires:       bluez
Requires:       bluez-tools
Requires:       polkit
Requires:       libnotify
Requires:       libayatana-appindicator-gtk3

%description
tws_manager is a Linux desktop client for controlling Nothing and CMF earbuds
over Bluetooth RFCOMM with tray support, autoconnect, and safe protocol tooling.

%prep
%setup -q

%build
go build -o bin/tws_manager ./cmd/tws_manager
go build -tags "gio systray" -o bin/tws_manager_gio ./cmd/tws_manager_gio
go build -o bin/tws_manager_rfcomm_helper ./cmd/tws_manager_rfcomm_helper

%install
install -Dpm0755 bin/tws_manager %{buildroot}%{_bindir}/tws_manager
install -Dpm0755 bin/tws_manager_gio %{buildroot}%{_bindir}/tws_manager_gio
install -Dpm0755 bin/tws_manager_rfcomm_helper %{buildroot}%{_libexecdir}/tws_manager_rfcomm_helper
install -Dpm0644 packaging/common/tws_manager.desktop %{buildroot}%{_datadir}/applications/tws_manager.desktop
install -Dpm0644 packaging/common/tws_manager-autostart.desktop %{buildroot}%{_sysconfdir}/xdg/autostart/tws_manager.desktop
install -Dpm0644 packaging/common/tws_manager.svg %{buildroot}%{_datadir}/icons/hicolor/scalable/apps/tws_manager.svg
install -Dpm0644 packaging/common/org.tws_manager.rfcomm.policy %{buildroot}%{_datadir}/polkit-1/actions/org.tws_manager.rfcomm.policy
install -Dpm0644 packaging/common/90-tws_manager.rules %{buildroot}%{_sysconfdir}/polkit-1/rules.d/90-tws_manager.rules
install -Dpm0644 packaging/common/tws_manager.sysusers %{buildroot}%{_sysusersdir}/tws_manager.conf
install -Dpm0644 README.md %{buildroot}%{_docdir}/tws_manager/README.md
install -Dpm0644 SECURITY.md %{buildroot}%{_docdir}/tws_manager/SECURITY.md

%post
%sysusers_create %{_sysusersdir}/tws_manager.conf
/usr/bin/getent group tws_manager >/dev/null 2>&1 || true
echo "tws_manager: for rootless mode, add your user to group 'tws_manager':"
echo "  sudo usermod -aG tws_manager <username>"
echo "Then log out and log back in."

%files
%{_bindir}/tws_manager
%{_bindir}/tws_manager_gio
%{_libexecdir}/tws_manager_rfcomm_helper
%{_datadir}/applications/tws_manager.desktop
%{_sysconfdir}/xdg/autostart/tws_manager.desktop
%{_datadir}/icons/hicolor/scalable/apps/tws_manager.svg
%{_datadir}/polkit-1/actions/org.tws_manager.rfcomm.policy
%{_sysconfdir}/polkit-1/rules.d/90-tws_manager.rules
%{_sysusersdir}/tws_manager.conf
%doc README.md
%doc SECURITY.md

%changelog
* Fri Jun 05 2026 tws_manager maintainers <maintainers@example.com> - 0.1.0-1
- Initial package with rootless polkit helper and desktop autostart.
