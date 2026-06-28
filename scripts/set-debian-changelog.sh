#!/usr/bin/env bash
# Write packaging/debian/changelog and validate it with dpkg-parsechangelog.
#
# Env:
#   PKG_VERSION / APP_VERSION  release label for the changelog bullet (default: pkg-version.sh)
#   DEBIAN_REVISION            Debian revision, default 1
#   DEBIAN_DIST                distribution, default unstable
#   DEBIAN_URGENCY             urgency, default medium
#   DEBIAN_PKG                 source/binary package name, default tws-manager
#   DEBIAN_CHANGELOG_MSG       override bullet text
#   SOURCE_DATE_EPOCH          reproducible timestamp (seconds since epoch)
set -euo pipefail

root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
pkg="${DEBIAN_PKG:-tws-manager}"
revision="${DEBIAN_REVISION:-1}"
distribution="${DEBIAN_DIST:-unstable}"
urgency="${DEBIAN_URGENCY:-medium}"
upstream="$("${root}/scripts/debian-version.sh")"
display="${PKG_VERSION:-${APP_VERSION:-${upstream}}}"
message="${DEBIAN_CHANGELOG_MSG:-Release ${display}.}"
changelog="${root}/packaging/debian/changelog"
maintainer_name="${DEBIAN_FULLNAME:-tws_manager maintainers}"
maintainer_email="${DEBIAN_EMAIL:-kuksyenok.i.s@gmail.com}"
maintainer="${maintainer_name} <${maintainer_email}>"
version="${upstream}-${revision}"

changelog_timestamp() {
	if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
		if date -R -u -d "@${SOURCE_DATE_EPOCH}" >/dev/null 2>&1; then
			date -R -u -d "@${SOURCE_DATE_EPOCH}"
			return
		fi
		if date -R -u -r "${SOURCE_DATE_EPOCH}" >/dev/null 2>&1; then
			date -R -u -r "${SOURCE_DATE_EPOCH}"
			return
		fi
	fi
	if git -C "${root}" rev-parse --verify HEAD >/dev/null 2>&1; then
		git -C "${root}" log -1 --format=%cD
		return
	fi
	date -R
}

timestamp="$(changelog_timestamp)"

if [[ ! "${pkg}" =~ ^[a-z0-9][a-z0-9.+~-]*$ ]] || [[ "${pkg}" == *"_"* ]]; then
	echo "error: invalid Debian package name: ${pkg} (use lowercase letters, digits, + - . ~ only)" >&2
	exit 1
fi

mkdir -p "$(dirname "${changelog}")"
{
	printf '%s (%s) %s; urgency=%s\n' "${pkg}" "${version}" "${distribution}" "${urgency}"
	printf '\n'
	printf '  * %s\n' "${message}"
	printf '\n'
	printf ' -- %s  %s\n' "${maintainer}" "${timestamp}"
} > "${changelog}"

if command -v dpkg-parsechangelog >/dev/null 2>&1; then
	parsed_version="$(dpkg-parsechangelog -l "${changelog}" -S Version 2>/dev/null || true)"
	if [[ -z "${parsed_version}" ]]; then
		echo "error: dpkg-parsechangelog rejected ${changelog}:" >&2
		dpkg-parsechangelog -l "${changelog}" 2>&1 >&2 || true
		sed -n '1,10p' "${changelog}" | cat -A >&2
		exit 1
	fi
	if [[ "${parsed_version}" != "${version}" ]]; then
		echo "error: parsed version ${parsed_version} != ${version}" >&2
		exit 1
	fi
fi

echo "Wrote ${changelog} (${pkg} ${version}, ${timestamp})"
