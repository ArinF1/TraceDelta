#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

printf 'Fuzzing OTLP parser input handling...\n'
go test ./internal/otlp -run '^$' -fuzz '^FuzzParse$' -fuzztime=2s

printf 'Fuzzing trace/span matching determinism...\n'
go test ./internal/match -run '^$' -fuzz '^FuzzTraceAndSpanMatchingIgnoresTraceOrder$' -fuzztime=2s

printf 'Fuzzing reporter escaping...\n'
go test ./internal/report -run '^$' -fuzz '^FuzzReportersEscapeTraceDerivedStrings$' -fuzztime=2s

printf 'TraceDelta bounded fuzz smoke passed.\n'
