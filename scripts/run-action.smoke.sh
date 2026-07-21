#!/usr/bin/env bash
set -u

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
binary="${1:-}"
if [[ -z "${binary}" || ! -x "${binary}" ]]; then
  printf 'usage: %s TRACEDELTA_BINARY\n' "$0" >&2
  exit 2
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/tracedelta-action-smoke.XXXXXX")"
cleanup() {
  rm -rf -- "${work_dir}"
}
trap cleanup EXIT

run_action() {
  output_file="$1"
  baseline="$2"
  candidate="$3"
  report_dir="$4"
  TRACEDELTA_ACTION_BINARY="${binary}" \
  TRACEDELTA_ACTION_BASELINE="${baseline}" \
  TRACEDELTA_ACTION_CANDIDATE="${candidate}" \
  TRACEDELTA_ACTION_REPORT_DIRECTORY="${report_dir}" \
  TRACEDELTA_ACTION_DURATION_THRESHOLD="20%" \
  TRACEDELTA_ACTION_DURATION_THRESHOLD_ABSOLUTE="10ms" \
  TRACEDELTA_ACTION_REDACT_ATTRIBUTES=$'custom.secret\n\n' \
  TRACEDELTA_ACTION_ROOT="${repo_root}" \
  GITHUB_OUTPUT="${output_file}" \
    "${BASH}" "${repo_root}/scripts/run-action.sh"
}

regression_output="${work_dir}/regression-output"
run_action \
  "${regression_output}" \
  "${repo_root}/testdata/baseline.json" \
  "${repo_root}/testdata/candidate.json" \
  "${work_dir}/regression-reports"
regression_status=$?
if [[ "${regression_status}" -ne 0 ]]; then
  printf 'Action smoke: regression result should leave the Action step successful.\n' >&2
  exit 1
fi
if ! grep -qx 'exit-code=1' "${regression_output}" || ! grep -qx 'outcome=differences' "${regression_output}"; then
  printf 'Action smoke: regression outputs are incorrect.\n' >&2
  exit 1
fi
if [[ ! -s "${work_dir}/regression-reports/tracedelta-report.txt" || ! -s "${work_dir}/regression-reports/tracedelta-report.json" || ! -s "${work_dir}/regression-reports/tracedelta-report.html" ]]; then
  printf 'Action smoke: one or more regression reports are missing.\n' >&2
  exit 1
fi

error_output="${work_dir}/error-output"
run_action \
  "${error_output}" \
  "${repo_root}/testdata/baseline.json" \
  "${work_dir}/missing.json" \
  "${work_dir}/error-reports"
error_status=$?
if [[ "${error_status}" -ne 2 ]]; then
  printf 'Action smoke: missing input should fail the Action step with exit 2.\n' >&2
  exit 1
fi
if ! grep -qx 'exit-code=2' "${error_output}" || ! grep -qx 'outcome=error' "${error_output}"; then
  printf 'Action smoke: tool-error outputs are incorrect.\n' >&2
  exit 1
fi
if [[ -e "${work_dir}/error-reports/tracedelta-report.txt" || -e "${work_dir}/error-reports/tracedelta-report.json" || -e "${work_dir}/error-reports/tracedelta-report.html" ]]; then
  printf 'Action smoke: tool error left a report artifact.\n' >&2
  exit 1
fi

printf 'TraceDelta Action entry-point smoke passed.\n'
