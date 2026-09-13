Name:           nothing_helper
Version:        0.1.0
Release:        1%{?dist}
Obsoletes:      tws_manager
Provides:       tws_manager = %{version}-%{release}
Summary:        Nothing_helper desktop client
License:        MIT
URL:            https://github.com/Kuksenok-i-s/nothing_helper
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
Nothing_helper is a Linux desktop client for controlling Nothing and CMF earbuds
over Bluetooth RFCOMM with tray support, autoconnect, and safe protocol tooling.

%prep
%setup -q

%build
go build -tags "gio systray" -o bin/nothing_helper ./cmd/nothing_helper
go build -o bin/nothing_helper_rfcomm_helper ./cmd/nothing_helper_rfcomm_helper

%install
install -Dpm0755 bin/nothing_helper %{buildroot}%{_bindir}/nothing_helper
install -Dpm0755 bin/nothing_helper_rfcomm_helper %{buildroot}%{_libexecdir}/nothing_helper_rfcomm_helper
install -Dpm0644 packaging/common/nothing_helper.desktop %{buildroot}%{_datadir}/applications/nothing_helper.desktop
install -Dpm0644 packaging/common/nothing_helper-autostart.desktop %{buildroot}%{_sysconfdir}/xdg/autostart/nothing_helper.desktop
install -Dpm0644 packaging/common/nothing_helper.svg %{buildroot}%{_datadir}/icons/hicolor/scalable/apps/nothing_helper.svg
install -Dpm0644 packaging/common/org.nothing_helper.rfcomm.policy %{buildroot}%{_datadir}/polkit-1/actions/org.nothing_helper.rfcomm.policy
install -Dpm0644 packaging/common/90-nothing_helper.rules %{buildroot}%{_sysconfdir}/polkit-1/rules.d/90-nothing_helper.rules
install -Dpm0644 packaging/common/nothing_helper.sysusers %{buildroot}%{_sysusersdir}/nothing_helper.conf
install -Dpm0644 README.md %{buildroot}%{_docdir}/nothing_helper/README.md
install -Dpm0644 SECURITY.md %{buildroot}%{_docdir}/nothing_helper/SECURITY.md

%post
%sysusers_create %{_sysusersdir}/nothing_helper.conf
/usr/bin/getent group nothing_helper >/dev/null 2>&1 || true
echo "nothing_helper: for rootless mode, add your user to group 'nothing_helper':"
echo "  sudo usermod -aG nothing_helper <username>"
echo "Then log out and log back in."

%files
%{_bindir}/nothing_helper
%{_libexecdir}/nothing_helper_rfcomm_helper
%{_datadir}/applications/nothing_helper.desktop
%{_sysconfdir}/xdg/autostart/nothing_helper.desktop
%{_datadir}/icons/hicolor/scalable/apps/nothing_helper.svg
%{_datadir}/polkit-1/actions/org.nothing_helper.rfcomm.policy
%{_sysconfdir}/polkit-1/rules.d/90-nothing_helper.rules
%{_sysusersdir}/nothing_helper.conf
%doc README.md
%doc SECURITY.md

%changelog
* Fri Jun 05 2026 nothing_helper maintainers <kuksyenok.i.s@gmail.com> - 0.1.0-1
- Initial package with rootless polkit helper and desktop autostart.
