#!/usr/bin/env bash
set -u

required_variables=(
  TRACEDELTA_ACTION_BINARY
  TRACEDELTA_ACTION_BASELINE
  TRACEDELTA_ACTION_CANDIDATE
  TRACEDELTA_ACTION_REPORT_DIRECTORY
  TRACEDELTA_ACTION_DURATION_THRESHOLD
  TRACEDELTA_ACTION_DURATION_THRESHOLD_ABSOLUTE
  TRACEDELTA_ACTION_REDACT_ATTRIBUTES
  TRACEDELTA_ACTION_ROOT
  GITHUB_OUTPUT
)
for variable_name in "${required_variables[@]}"; do
  if [[ ! -v "${variable_name}" ]]; then
    printf 'TraceDelta Action: required environment variable is missing: %s\n' "${variable_name}" >&2
    exit 2
  fi
done

if [[ "${TRACEDELTA_ACTION_REPORT_DIRECTORY}" == *$'\n'* || "${TRACEDELTA_ACTION_REPORT_DIRECTORY}" == *$'\r'* ]]; then
  printf 'TraceDelta Action: report-directory must not contain a newline.\n' >&2
  exit 2
fi

compare_options=(
  --duration-threshold "${TRACEDELTA_ACTION_DURATION_THRESHOLD}"
  --duration-threshold-absolute "${TRACEDELTA_ACTION_DURATION_THRESHOLD_ABSOLUTE}"
)
while IFS= read -r attribute_key || [[ -n "${attribute_key}" ]]; do
  if [[ "${attribute_key}" =~ ^[[:space:]]*$ ]]; then
    continue
  fi
  compare_options+=(--redact-attribute "${attribute_key}")
done <<< "${TRACEDELTA_ACTION_REDACT_ATTRIBUTES}"

"${BASH}" "${TRACEDELTA_ACTION_ROOT}/scripts/ci-compare.sh" \
  "${TRACEDELTA_ACTION_BINARY}" \
  "${TRACEDELTA_ACTION_BASELINE}" \
  "${TRACEDELTA_ACTION_CANDIDATE}" \
  "${TRACEDELTA_ACTION_REPORT_DIRECTORY}" \
  "${compare_options[@]}"
comparison_status=$?

text_report="${TRACEDELTA_ACTION_REPORT_DIRECTORY}/tracedelta-report.txt"
if [[ "${comparison_status}" -eq 0 || "${comparison_status}" -eq 1 ]]; then
  "${TRACEDELTA_ACTION_BINARY}" compare \
    --baseline "${TRACEDELTA_ACTION_BASELINE}" \
    --candidate "${TRACEDELTA_ACTION_CANDIDATE}" \
    "${compare_options[@]}" \
    --format text \
    --output "${text_report}"
  text_status=$?
  if [[ ("${text_status}" -ne 0 && "${text_status}" -ne 1) || "${text_status}" -ne "${comparison_status}" || ! -s "${text_report}" ]]; then
    rm -f -- \
      "${TRACEDELTA_ACTION_REPORT_DIRECTORY}/tracedelta-report.json" \
      "${TRACEDELTA_ACTION_REPORT_DIRECTORY}/tracedelta-report.html" \
      "${text_report}"
    printf 'TraceDelta Action: terminal report failed or disagreed with the JSON/HTML result; no reports were preserved.\n' >&2
    comparison_status=2
  fi
fi

case "${comparison_status}" in
  0)
    outcome="no_differences"
    ;;
  1)
    outcome="differences"
    ;;
  *)
    comparison_status=2
    outcome="error"
    ;;
esac

{
  printf 'exit-code=%d\n' "${comparison_status}"
  printf 'outcome=%s\n' "${outcome}"
  printf 'report-directory=%s\n' "${TRACEDELTA_ACTION_REPORT_DIRECTORY}"
  printf 'text-report=%s\n' "${text_report}"
  printf 'json-report=%s/tracedelta-report.json\n' "${TRACEDELTA_ACTION_REPORT_DIRECTORY}"
  printf 'html-report=%s/tracedelta-report.html\n' "${TRACEDELTA_ACTION_REPORT_DIRECTORY}"
} >> "${GITHUB_OUTPUT}"

if [[ "${comparison_status}" -eq 2 ]]; then
  exit 2
fi
exit 0
