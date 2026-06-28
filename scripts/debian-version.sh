#!/usr/bin/env bash
# Normalize an upstream version for Debian packaging.
# Debian numeric components must not contain leading zeros (1.03 -> 1.3).
set -euo pipefail

version="${1:-${APP_VERSION:-${PKG_VERSION:-}}}"
if [[ -z "${version}" ]]; then
	version="$(CDPATH= cd -- "$(dirname "$0")/.." && ./scripts/pkg-version.sh)"
fi

epoch=""
rest="${version}"
if [[ "${rest}" == *:* ]]; then
	epoch="${rest%%:*}:"
	rest="${rest#*:}"
fi

prefix="${rest}"
suffix=""
if [[ "${rest}" == *~* ]]; then
	prefix="${rest%%~*}"
	suffix="~${rest#*~}"
fi

IFS='.' read -r -a parts <<< "${prefix}"
normalized=()
for part in "${parts[@]}"; do
	if [[ "${part}" =~ ^[0-9]+$ ]]; then
		normalized+=("$((10#${part}))")
	else
		normalized+=("${part}")
	fi
done

IFS='.'
printf '%s%s%s\n' "${epoch}" "${normalized[*]}" "${suffix}"
