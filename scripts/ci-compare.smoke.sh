#!/usr/bin/env bash
set -u

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
binary="${1:-}"
wrapper="${repo_root}/scripts/ci-compare.sh"
shell_binary="${BASH}"

if [[ -z "${binary}" || ! -x "${binary}" ]]; then
  printf 'usage: %s TRACEDELTA_BINARY\n' "$0" >&2
  exit 2
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/tracedelta-ci-smoke.XXXXXX")"
cleanup() {
  rm -rf -- "${work_dir}"
}
trap cleanup EXIT

expect_status() {
  expected="$1"
  shift
  "$@"
  actual=$?
  if [[ "${actual}" -ne "${expected}" ]]; then
    printf 'CI smoke: expected exit %d, got %d: %s\n' "${expected}" "${actual}" "$*" >&2
    exit 1
  fi
}

assert_reports() {
  report_dir="$1"
  json_report="${report_dir}/tracedelta-report.json"
  html_report="${report_dir}/tracedelta-report.html"
  if [[ ! -s "${json_report}" || ! -s "${html_report}" ]]; then
    printf 'CI smoke: expected non-empty JSON and HTML reports in %s\n' "${report_dir}" >&2
    exit 1
  fi
  if ! grep -q '"schemaVersion": "tracedelta.report/v1"' "${json_report}"; then
    printf 'CI smoke: JSON report has the wrong schema marker.\n' >&2
    exit 1
  fi
  if ! grep -qi '<!doctype html>' "${html_report}"; then
    printf 'CI smoke: HTML report is not standalone HTML.\n' >&2
    exit 1
  fi
}

pass_dir="${work_dir}/pass-reports"
expect_status 0 "${shell_binary}" "${wrapper}" "${binary}" \
  "${repo_root}/testdata/baseline.json" \
  "${repo_root}/testdata/baseline.json" \
  "${pass_dir}"
assert_reports "${pass_dir}"

regression_dir="${work_dir}/regression-reports"
expect_status 1 "${shell_binary}" "${wrapper}" "${binary}" \
  "${repo_root}/testdata/baseline.json" \
  "${repo_root}/testdata/candidate.json" \
  "${regression_dir}"
assert_reports "${regression_dir}"

invalid_trace="${work_dir}/invalid.json"
printf '{ invalid JSON\n' > "${invalid_trace}"
invalid_dir="${work_dir}/invalid-reports"
expect_status 2 "${shell_binary}" "${wrapper}" "${binary}" \
  "${repo_root}/testdata/baseline.json" \
  "${invalid_trace}" \
  "${invalid_dir}"
if [[ -e "${invalid_dir}/tracedelta-report.json" || -e "${invalid_dir}/tracedelta-report.html" ]]; then
  printf 'CI smoke: invalid input left a report artifact.\n' >&2
  exit 1
fi

missing_dir="${work_dir}/missing-reports"
expect_status 2 "${shell_binary}" "${wrapper}" "${binary}" \
  "${repo_root}/testdata/baseline.json" \
  "${work_dir}/missing.json" \
  "${missing_dir}"
if [[ -e "${missing_dir}" ]]; then
  printf 'CI smoke: missing input created a report directory.\n' >&2
  exit 1
fi

printf 'TraceDelta generic CI smoke passed.\n'
