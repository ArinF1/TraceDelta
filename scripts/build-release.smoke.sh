#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/tracedelta-release-smoke.XXXXXX")"
cleanup() {
  rm -rf -- "${work_dir}"
}
trap cleanup EXIT

version="v0.1.0-test"
version_name="${version#v}"
output_dir="${work_dir}/dist"
"${BASH}" "${repo_root}/scripts/build-release.sh" "${version}" "${output_dir}"

expected=(
  "tracedelta_${version_name}_linux_amd64"
  "tracedelta_${version_name}_linux_arm64"
  "tracedelta_${version_name}_darwin_amd64"
  "tracedelta_${version_name}_darwin_arm64"
  "tracedelta_${version_name}_windows_amd64.exe"
  "tracedelta_${version_name}_checksums.txt"
)
for name in "${expected[@]}"; do
  if [[ ! -s "${output_dir}/${name}" ]]; then
    printf 'Release smoke: missing or empty artifact: %s\n' "${name}" >&2
    exit 1
  fi
done
if [[ "$(find "${output_dir}" -maxdepth 1 -type f | wc -l | tr -d ' ')" -ne 6 ]]; then
  printf 'Release smoke: output contains an unexpected file count.\n' >&2
  exit 1
fi
(
  cd "${output_dir}"
  sha256sum --check "tracedelta_${version_name}_checksums.txt"
)

case "$(uname -m)" in
  x86_64|amd64) native_arch="amd64" ;;
  arm64|aarch64) native_arch="arm64" ;;
  *) native_arch="" ;;
esac
case "$(uname -s)" in
  Linux*) native_binary="${output_dir}/tracedelta_${version_name}_linux_${native_arch}" ;;
  Darwin*) native_binary="${output_dir}/tracedelta_${version_name}_darwin_${native_arch}" ;;
  MINGW*|MSYS*|CYGWIN*)
    if [[ "${native_arch}" == "amd64" ]]; then
      native_binary="${output_dir}/tracedelta_${version_name}_windows_amd64.exe"
    else
      native_binary=""
    fi
    ;;
  *) native_binary="" ;;
esac
if [[ -n "${native_binary}" && ! -f "${native_binary}" ]]; then
  native_binary=""
fi
if [[ -n "${native_binary}" ]]; then
  "${native_binary}" compare \
    --baseline "${repo_root}/testdata/baseline.json" \
    --candidate "${repo_root}/testdata/baseline.json" \
    > /dev/null
fi

if "${BASH}" "${repo_root}/scripts/build-release.sh" "${version}" "${output_dir}"; then
  printf 'Release smoke: existing output directory was unexpectedly replaced.\n' >&2
  exit 1
fi
if "${BASH}" "${repo_root}/scripts/build-release.sh" "not-a-version" "${work_dir}/invalid"; then
  printf 'Release smoke: invalid version was unexpectedly accepted.\n' >&2
  exit 1
fi

printf 'TraceDelta five-target release smoke passed.\n'
