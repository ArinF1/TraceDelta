#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

build_dir="$(mktemp -d "${TMPDIR:-/tmp}/tracedelta-check.XXXXXX")"
cleanup() {
  rm -rf -- "${build_dir}"
}
trap cleanup EXIT

printf 'Checking gofmt...\n'
unformatted="$(find . -type f -name '*.go' ! -path './vendor/*' -exec gofmt -l {} + | sort)"
if [[ -n "${unformatted}" ]]; then
  printf 'The following files require gofmt:\n%s\n' "${unformatted}" >&2
  exit 1
fi

printf 'Running tests...\n'
go test ./...

printf 'Running go vet...\n'
go vet ./...

printf 'Building CLI...\n'
go build -trimpath -o "${build_dir}/tracedelta" ./cmd/tracedelta

printf 'All essential checks passed.\n'
