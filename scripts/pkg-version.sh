#!/usr/bin/env bash
# Resolve release version from tag, workflow input, or dev snapshot.
set -euo pipefail

if [[ -n "${APP_VERSION:-}" ]]; then
	printf '%s\n' "${APP_VERSION}"
elif [[ "${GITHUB_REF:-}" == refs/tags/v* ]]; then
	printf '%s\n' "${GITHUB_REF#refs/tags/v}"
else
	sha="${GITHUB_SHA:-$(git rev-parse --short=7 HEAD 2>/dev/null || echo local)}"
	printf '0.0.0~dev.%s\n' "${sha:0:7}"
fi
