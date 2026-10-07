# qyvora-common

**Status:** `CURRENT` · **Not a security tool**
**Last verified against implementation:** 2026-10-05

`qyvora-common` is the **reference machine contract** and **conformance
harness** for the fourteen QYVORA security frameworks.

It is one of three supporting repositories alongside `qyvora-tui` (shared
terminal UI) and `qyvora-dist` (distribution). **None of the three is a
security framework.** Counting any of them as a tool makes the ecosystem
"15 tools", which is wrong.

## What this module is

| | |
|---|---|
| Module path | `github.com/QYVORA/qyvora-common` |
| Go | `1.26` |
| Packages | `contract/`, `conformance/`, `cmd/qyvora-conformance` |
| Imported by the frameworks? | **No.** Each framework implements the contract natively. |

This is deliberate. The frameworks stay independent Go modules; `contract/` is
the *authored schema* they are all measured against, not a library they link
against. See the package doc in `contract/contract.go`.

## Contents

```
contract/contract.go          authored schema + exit codes + validators
contract/contract_test.go     validator tests
 conformance/runner.go         builds a framework and probes its real binary
 conformance/tui_test.go       `go version -m` build-info parsing for TUI linkage
 cmd/qyvora-conformance/main.go  the CLI
```

### `contract/` — the schema

- `Identity` — the identity block carried by every machine stream and result
- `Envelope` — the 7-field JSONL event envelope: `schema_version`, `timestamp`,
  `execution_id`, `framework`, `level`, `event`, `data`
- `Context` — ties one run to a result, a target, and an optional session
- `Target` — the object an assessment runs against (type/value/authorization)
- `Command` — one capability record an orchestrator can call without guessing
- `CapabilityManifest` — the machine-readable capability list
- `VersionResult`, `Result`, `FindFinding`, `Risk` — canonical result shapes

**Exit-code contract** (`contract/contract.go:18-24`):

| Constant | Code | Meaning |
|---|---|---|
| `ExitSuccess` | 0 | success |
| `ExitRuntime` | 1 | runtime / general error |
| `ExitUsage` | 2 | usage error |
| `ExitUnsupported` | 3 | authorization-decline / domain-unsupported, by convention |
| `ExitInterrupted` | 130 | interrupted (`128 + SIGINT`) |

Validators: `IsSemverVersion`, `ValidateVersion` (a framework must not report
`dev`), `ValidateExitCode`.

### `conformance/` — the runner

`conformance.Workspace` discovers the frameworks on disk; `Build` compiles one;
`Run` probes the **real built binary** — not the source — and checks:

- `version` machine output parses as JSON, names the right framework, and
  reports a semantic version (never `dev`);
- an unknown **flag** exits `2` (usage) rather than `1` (runtime);
- an unknown **command** exits `2` where the CLI distinguishes commands from
  target positionals. This probe is **skipped for Anansi**, whose root accepts a
  positional target (`anansi example.com`);
- the `capabilities` command, when advertised, succeeds. An absent command is
  recorded as a documented gap, not a failure.
- the release binary actually **links** `github.com/QYVORA/qyvora-tui`, proved by
  the `h1:` content hash the linker records in `go version -m` output. A
  `require` alone does not pass this check, which is exactly what catches a
  framework whose TUI import was dropped or moved to a build tag.

The runner exits non-zero when any check fails, so it can run in CI.

## Usage

```bash
# one framework
go run ./cmd/qyvora-conformance -root .. -bin /tmp/qf-shaka:shaka

# all frameworks in a workspace (builds each one)
go run ./cmd/qyvora-conformance -root ..

# machine-readable results
go run ./cmd/qyvora-conformance -root .. -json
```

`-skip-build` reuses already-built binaries, and each `-bin` entry is
`<path>:<name>`.

## Known gaps (verified 2026-10-05)

- **Aksum, Anansi, and TOHA3EE do not implement a `capabilities` command.** The
  runner records these as documented gaps rather than failures, so conformance
  passes while the capability gap persists. `IN PROGRESS`.
  - Anansi must **not** be given a `capabilities` probe: its root command treats
    any unrecognized positional as a scan target, so `anansi capabilities` runs a
    real scan against a host literally named `capabilities` (reaching out to
    crt.sh) instead of printing a manifest.
- Each framework's `scripts/verify-artifact.sh` repeats the TUI linkage check
  against the **released** binary, so a published artifact cannot silently ship
  without its TUI.

## Ecosystem position

Last full run: **14/14 frameworks fully conformant**, `0` warnings, as part of
`./check-all.sh --quick` (2026-10-05). `check-all.sh` builds and tests this
module too, but does **not** cover `qyvora-tui`.

**Capability coverage validation:** For PROMPT.md compliance checking (output formats,
tiers, noise levels), see the standalone `qyvora-conformance` tool. The two
conformance validators serve different phases:
- This tool (qyvora-common) → Machine contract for Phase 5 (AI orchestration)
- qyvora-conformance → Capability requirements for Phase 1 (standardization)

Related: the normative output contract is
[`QYVORA-TOOL-OUTPUT-SPEC.md`](../../../knowledge/qyvora-docs/09-technical/cross-project/QYVORA-TOOL-OUTPUT-SPEC.md).
