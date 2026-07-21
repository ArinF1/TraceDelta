#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf 'usage: %s VERSION OUTPUT_DIRECTORY\n' "$0" >&2
}

if [[ "$#" -ne 2 ]]; then
  usage
  exit 2
fi

version="$1"
output_dir="$2"
if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  printf 'TraceDelta release: version must look like v0.1.0 or v0.1.0-rc.1: %s\n' "${version}" >&2
  exit 2
fi
if [[ -e "${output_dir}" ]]; then
  printf 'TraceDelta release: output directory already exists: %s\n' "${output_dir}" >&2
  exit 2
fi

output_parent="$(dirname -- "${output_dir}")"
if [[ ! -d "${output_parent}" ]]; then
  printf 'TraceDelta release: output parent does not exist: %s\n' "${output_parent}" >&2
  exit 2
fi

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
staging_dir="$(mktemp -d "${output_parent}/.tracedelta-release.XXXXXX")"
cleanup() {
  rm -rf -- "${staging_dir}"
}
trap cleanup EXIT

version_name="${version#v}"
targets=(
  linux/amd64
  linux/arm64
  darwin/amd64
  darwin/arm64
  windows/amd64
)
artifact_names=()

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  extension=""
  if [[ "${goos}" == "windows" ]]; then
    extension=".exe"
  fi
  artifact_name="tracedelta_${version_name}_${goos}_${goarch}${extension}"
  artifact_names+=("${artifact_name}")
  printf 'Building %s/%s...\n' "${goos}" "${goarch}"
  (
    cd "${repo_root}"
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
      go build -buildvcs=false -trimpath -o "${staging_dir}/${artifact_name}" ./cmd/tracedelta
  )
done

checksum_name="tracedelta_${version_name}_checksums.txt"
(
  cd "${staging_dir}"
  LC_ALL=C sha256sum "${artifact_names[@]}" > "${checksum_name}"
)

mv -- "${staging_dir}" "${output_dir}"
printf 'TraceDelta release artifacts: %s\n' "${output_dir}"
