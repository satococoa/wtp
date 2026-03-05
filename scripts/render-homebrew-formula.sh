#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

VERSION=""
SHA256=""
TEMPLATE_PATH="${REPO_ROOT}/packaging/homebrew/wtp.rb.tmpl"
OUTPUT_PATH=""

usage() {
  cat <<'EOF'
Usage:
  scripts/render-homebrew-formula.sh --version <VERSION> --sha256 <SHA256> [--template <PATH>] [--output <PATH>]

Examples:
  scripts/render-homebrew-formula.sh --version 2.8.0 --sha256 <sha>
  scripts/render-homebrew-formula.sh --version 2.8.0 --sha256 <sha> --output /tmp/wtp.rb
EOF
}

require_option_value() {
  local option="$1"
  local value="${2:-}"

  if [[ -z "${value}" || "${value}" == --* ]]; then
    echo "missing value for ${option}" >&2
    usage >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      require_option_value "$1" "${2:-}"
      VERSION="$2"
      shift 2
      ;;
    --sha256)
      require_option_value "$1" "${2:-}"
      SHA256="$2"
      shift 2
      ;;
    --template)
      require_option_value "$1" "${2:-}"
      TEMPLATE_PATH="$2"
      shift 2
      ;;
    --output)
      require_option_value "$1" "${2:-}"
      OUTPUT_PATH="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "${VERSION}" || -z "${SHA256}" ]]; then
  echo "--version and --sha256 are required" >&2
  usage >&2
  exit 1
fi

if [[ ! -f "${TEMPLATE_PATH}" ]]; then
  echo "template not found: ${TEMPLATE_PATH}" >&2
  exit 1
fi

rendered="$(
  sed \
    -e "s/__VERSION__/${VERSION}/g" \
    -e "s/__SHA256__/${SHA256}/g" \
    "${TEMPLATE_PATH}"
)"

if grep -q '__VERSION__\|__SHA256__' <<<"${rendered}"; then
  echo "render failed: unresolved placeholders in output" >&2
  exit 1
fi

if [[ -n "${OUTPUT_PATH}" ]]; then
  printf '%s\n' "${rendered}" > "${OUTPUT_PATH}"
else
  printf '%s\n' "${rendered}"
fi
