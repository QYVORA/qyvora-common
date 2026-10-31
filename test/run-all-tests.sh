#!/usr/bin/env bash
#
# Local qyvora-common test suite.
#
#   --fast          gofmt/vet/build/unit tests
#   --full          also exercise the capvalidator CLI with fake frameworks
#   --keep          keep temporary artifacts
#   --verbose       show command output
#   --no-color      disable ANSI colour
#
# Exit codes: 0 pass, 1 test failure, 2 usage error.

set -uo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname -- "$HERE")"
MODE="full"
KEEP=0
VERBOSE=0
NO_COLOR=0

usage() { sed -n '2,11p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; }
while [ $# -gt 0 ]; do
    case "$1" in
        --fast) MODE="fast" ;;
        --full) MODE="full" ;;
        --keep) KEEP=1 ;;
        --verbose|-v) VERBOSE=1 ;;
        --no-color) NO_COLOR=1 ;;
        -h|--help) usage; exit 0 ;;
        *) printf 'unknown option: %s\n' "$1" >&2; exit 2 ;;
    esac
    shift
done

if ! command -v go >/dev/null 2>&1; then
    printf 'go is required\n' >&2
    exit 1
fi
export GOPRIVATE="${GOPRIVATE:-github.com/QYVORA/*}"
export GOPROXY="${GOPROXY:-direct}"

GRN=''; RED=''; YEL=''; BLD=''; OFF=''
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "$NO_COLOR" -eq 0 ]; then
    GRN=$(printf '\033[32m'); RED=$(printf '\033[31m'); YEL=$(printf '\033[33m')
    BLD=$(printf '\033[1m'); OFF=$(printf '\033[0m')
fi

ART="$(mktemp -d "${TMPDIR:-/tmp}/qyvora-common-test.XXXXXX")" || exit 1
PASSED=0
FAILED=0
SKIPPED=0

cleanup() {
    if [ "$KEEP" -eq 1 ] || [ "$FAILED" -ne 0 ]; then
        printf '\nartifacts: %s\n' "$ART"
    else
        rm -rf "$ART"
    fi
}
trap cleanup EXIT

pass() { PASSED=$((PASSED + 1)); printf '  %sok%s   %s\n' "$GRN" "$OFF" "$1"; }
skip() { SKIPPED=$((SKIPPED + 1)); printf '  skip %s\n' "$1"; }
fail() { FAILED=$((FAILED + 1)); printf '  %sFAIL%s %s\n' "$RED" "$OFF" "$1"; }

show_log() {
    [ "$VERBOSE" -eq 1 ] || return 0
    [ -s "$1" ] || return 0
    sed 's/^/       /' "$1"
}

run_check() {
    local name="$1" log="$2"
    shift 2
    if "$@" >"$log" 2>&1; then
        pass "$name"
        show_log "$log"
        return 0
    fi
    fail "$name"
    show_log "$log"
    return 1
}

expect_exit() {
    local want="$1" name="$2" log="$3"
    shift 3
    "$@" >"$log" 2>&1
    local got=$?
    if [ "$got" -eq "$want" ]; then
        pass "$name"
        show_log "$log"
        return 0
    fi
    fail "$name (want exit $want, got $got)"
    show_log "$log"
    return 1
}

fake_framework() {
    local root="$1" name="$2" mode="$3"
    mkdir -p "$root/qyvora-$name/bin"
    for f in LICENSE NOTICE SECURITY.md README.md; do
        printf '%s\n' "$f" >"$root/qyvora-$name/$f"
    done
    local body
    if [ "$mode" = "valid" ]; then
        body='{"capabilities":[{"name":"demo","tier":1,"noise":"passive"}]}'
    else
        body='not-json'
    fi
    cat >"$root/qyvora-$name/bin/$name" <<SH
#!/usr/bin/env bash
case "\${1:-}" in
    version) exit 0 ;;
    capabilities) printf '%s\n' '$body'; exit 0 ;;
    --help) printf 'usage: $name\n'; exit 0 ;;
    *) exit 2 ;;
esac
SH
    chmod +x "$root/qyvora-$name/bin/$name"
}

printf '%sQYVORA common test suite%s\n\n' "$BLD" "$OFF"
printf '%s-- self-check --%s\n' "$YEL" "$OFF"

fmt_out="$(cd "$REPO" && gofmt -l . 2>&1)"
if [ -z "$fmt_out" ]; then pass "gofmt"; else fail "gofmt"; printf '%s\n' "$fmt_out" | sed 's/^/       /'; fi
run_check "go vet" "$ART/vet.log" bash -c "cd '$REPO' && go vet ./..."
run_check "go build" "$ART/build.log" bash -c "cd '$REPO' && go build ./..."
run_check "unit tests" "$ART/test.log" bash -c "cd '$REPO' && go test ./... -count=1"

if [ "$MODE" = "full" ]; then
    printf '\n%s-- capvalidator CLI --%s\n' "$YEL" "$OFF"
    CAP="$ART/bin/qyvora-capvalidator"
    if run_check "build capvalidator" "$ART/cap-build.log" \
        bash -c "cd '$REPO' && go build -o '$CAP' ./cmd/qyvora-capvalidator"; then
        fake_framework "$ART" anansi valid
        fake_framework "$ART" jabari invalid

        expect_exit 0 "valid framework passes" "$ART/cap-valid.log" \
            "$CAP" --tool anansi
        if grep -q 'Summary: 9/9 checks passed' "$ART/cap-valid.log"; then
            pass "valid framework reports 9/9 checks"
        else
            fail "valid framework reports 9/9 checks"
            show_log "$ART/cap-valid.log"
        fi

        expect_exit 1 "invalid capabilities JSON fails" "$ART/cap-invalid.log" \
            "$CAP" --tool jabari
        if grep -q 'capabilities_json_valid' "$ART/cap-invalid.log" || \
           grep -q 'jabari' "$ART/cap-invalid.log"; then
            pass "invalid framework is reported"
        else
            fail "invalid framework is reported"
            show_log "$ART/cap-invalid.log"
        fi

        expect_exit 1 "missing framework binary fails" "$ART/cap-missing.log" \
            "$CAP" --tool mansa
        expect_exit 2 "unknown framework is a usage error" "$ART/cap-unknown.log" \
            "$CAP" --tool does-not-exist

        expect_exit 0 "gap report mode" "$ART/cap-report.log" \
            "$CAP" --tool anansi --report
        if grep -q '^# QYVORA Capability Gap Report' "$ART/cap-report.log"; then
            pass "gap report has the expected heading"
        else
            fail "gap report has the expected heading"
            show_log "$ART/cap-report.log"
        fi
    else
        skip "capvalidator CLI checks (build failed)"
    fi
fi

printf '\n%s================================%s\n' "$YEL" "$OFF"
printf '  passed: %d  failed: %d  skipped: %d\n' "$PASSED" "$FAILED" "$SKIPPED"
if [ "$FAILED" -ne 0 ]; then
    printf '  %sFAILED%s\n' "$RED" "$OFF"
    exit 1
fi
printf '  %sAll common tests passed.%s\n' "$GRN" "$OFF"
exit 0
