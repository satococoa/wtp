#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TAP_REPO="${TAP_REPO:-satococoa/homebrew-tap}"
FORMULA_FILE="${FORMULA_FILE:-wtp.rb}"
TARGET_BRANCH="${TARGET_BRANCH:-main}"
SYNC_COMMIT_MESSAGE="${SYNC_COMMIT_MESSAGE:-chore: sync wtp Homebrew formula template}"
TOKEN="${HOMEBREW_TAP_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"

if [[ -z "${TOKEN}" ]]; then
  echo "HOMEBREW_TAP_GITHUB_TOKEN (or GITHUB_TOKEN) is required" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

tap_dir="${tmpdir}/tap"
formula_path="${tap_dir}/${FORMULA_FILE}"

git clone --depth=1 "https://x-access-token:${TOKEN}@github.com/${TAP_REPO}.git" "${tap_dir}"

if [[ ! -f "${formula_path}" ]]; then
  echo "formula not found in tap: ${FORMULA_FILE}" >&2
  exit 1
fi

version="$(sed -n 's/^  version "\([^"]*\)".*$/\1/p' "${formula_path}" | head -n1)"
sha256="$(sed -n 's/^  sha256 "\([^"]*\)".*$/\1/p' "${formula_path}" | head -n1)"

if [[ -z "${version}" || -z "${sha256}" ]]; then
  echo "failed to parse version/sha256 from ${FORMULA_FILE}" >&2
  exit 1
fi

"${SCRIPT_DIR}/render-homebrew-formula.sh" \
  --version "${version}" \
  --sha256 "${sha256}" \
  --output "${formula_path}"

if git -C "${tap_dir}" diff --quiet -- "${FORMULA_FILE}"; then
  echo "no Homebrew formula sync changes detected"
  exit 0
fi

git -C "${tap_dir}" config user.name "${GIT_AUTHOR_NAME:-wtp-bot}"
git -C "${tap_dir}" config user.email "${GIT_AUTHOR_EMAIL:-bot@satococoa.dev}"

git -C "${tap_dir}" add "${FORMULA_FILE}"
git -C "${tap_dir}" commit -m "${SYNC_COMMIT_MESSAGE}"
git -C "${tap_dir}" push origin "HEAD:${TARGET_BRANCH}"
