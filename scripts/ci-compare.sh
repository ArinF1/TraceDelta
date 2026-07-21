#!/usr/bin/env bash
set -u

usage() {
  printf 'usage: %s TRACEDELTA_BINARY BASELINE CANDIDATE REPORT_DIRECTORY [COMPARE_OPTION VALUE ...]\n' "$0" >&2
}

if [[ "$#" -lt 4 ]]; then
  usage
  exit 2
fi

binary="$1"
baseline="$2"
candidate="$3"
report_dir="$4"
shift 4

compare_options=()
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --duration-threshold|--duration-threshold-absolute|--redact-attribute)
      if [[ "$#" -lt 2 ]]; then
        printf 'TraceDelta CI: option requires a value: %s\n' "$1" >&2
        exit 2
      fi
      compare_options+=("$1" "$2")
      shift 2
      ;;
    *)
      printf 'TraceDelta CI: unsupported compare option: %s\n' "$1" >&2
      exit 2
      ;;
  esac
done

if [[ ! -x "${binary}" ]]; then
  printf 'TraceDelta CI: binary is missing or not executable: %s\n' "${binary}" >&2
  exit 2
fi
if [[ ! -f "${baseline}" ]]; then
  printf 'TraceDelta CI: baseline artifact is missing or not a file: %s\n' "${baseline}" >&2
  exit 2
fi
if [[ ! -f "${candidate}" ]]; then
  printf 'TraceDelta CI: candidate artifact is missing or not a file: %s\n' "${candidate}" >&2
  exit 2
fi
if ! mkdir -- "${report_dir}"; then
  printf 'TraceDelta CI: report directory must be a new path with an existing parent: %s\n' "${report_dir}" >&2
  exit 2
fi

json_report="${report_dir}/tracedelta-report.json"
html_report="${report_dir}/tracedelta-report.html"

remove_partial_reports() {
  rm -f -- "${json_report}" "${html_report}"
}

"${binary}" compare \
  --baseline "${baseline}" \
  --candidate "${candidate}" \
  "${compare_options[@]}" \
  --format json \
  --output "${json_report}"
json_status=$?

if [[ "${json_status}" -ne 0 && "${json_status}" -ne 1 ]]; then
  remove_partial_reports
  printf 'TraceDelta CI: comparison failed while producing JSON (exit %d); no reports were preserved.\n' "${json_status}" >&2
  exit 2
fi
if [[ ! -s "${json_report}" ]]; then
  remove_partial_reports
  printf 'TraceDelta CI: JSON report was not created; no reports were preserved.\n' >&2
  exit 2
fi

"${binary}" compare \
  --baseline "${baseline}" \
  --candidate "${candidate}" \
  "${compare_options[@]}" \
  --format html \
  --output "${html_report}"
html_status=$?

if [[ "${html_status}" -ne 0 && "${html_status}" -ne 1 ]]; then
  remove_partial_reports
  printf 'TraceDelta CI: comparison failed while producing HTML (exit %d); no reports were preserved.\n' "${html_status}" >&2
  exit 2
fi
if [[ ! -s "${html_report}" ]]; then
  remove_partial_reports
  printf 'TraceDelta CI: HTML report was not created; no reports were preserved.\n' >&2
  exit 2
fi
if [[ "${json_status}" -ne "${html_status}" ]]; then
  remove_partial_reports
  printf 'TraceDelta CI: report runs disagreed on the comparison result; no reports were preserved.\n' >&2
  exit 2
fi

printf 'TraceDelta CI: JSON report: %s\n' "${json_report}"
printf 'TraceDelta CI: HTML report: %s\n' "${html_report}"
if [[ "${json_status}" -eq 1 ]]; then
  printf 'TraceDelta CI: behavioral differences detected.\n'
else
  printf 'TraceDelta CI: no behavioral differences detected.\n'
fi
exit "${json_status}"
