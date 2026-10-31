# qyvora-common

**Status:** `CURRENT` · **Not a security tool**
**Version:** `v0.2.0`
**Last verified against implementation:** 2026-10-31

`qyvora-common` is the reference machine contract and capability validator for
the fourteen QYVORA security frameworks. The behavioral conformance runner now
lives in the standalone [`qyvora-conformance`](../qyvora-conformance) module.

It is one of four supporting repositories alongside `qyvora-tui` (shared
terminal UI), `qyvora-script` (shared scripting engine) and `qyvora-dist`
(distribution). **None of them is a security framework.**

## What this module is

| | |
|---|---|
| Module path | `github.com/QYVORA/qyvora-common` |
| Go | `1.26` |
| Packages | `contract/`, `capvalidator/`, `cmd/qyvora-capvalidator` |
| Imported by the frameworks? | **No.** Each framework implements the contract natively. |

`contract/` is the authored schema the frameworks are measured against. It is
not linked into framework binaries. `qyvora-conformance` imports it as the
reference schema for behavioral checks.

## Contents

```text
contract/contract.go               authored schema + exit codes + validators
contract/contract_test.go          validator tests
capvalidator/README.md             capability coverage validation documentation
cmd/qyvora-capvalidator/main.go    capability coverage validator CLI
```

### `contract/` — the schema

- `Identity` — identity block carried by machine streams and results.
- `Envelope` — canonical JSONL event envelope. It mirrors
  `qyvora-models.Event`, including `event_type`, `tier`, `module`, `action`,
  `severity`, `target`, `session_id` and `run_id`.
- `Context` — ties one run to a result, target and optional session.
- `Target` — object under assessment (type/value/authorization).
- `Command` — one capability record an orchestrator can call.
- `CapabilityManifest` — machine-readable capability list.
- `VersionResult`, `Result`, `FindFinding`, `Risk` — canonical result shapes.

**Exit-code contract:**

| Constant | Code | Meaning |
|---|---|---|
| `ExitSuccess` | 0 | success |
| `ExitRuntime` | 1 | runtime / general error |
| `ExitUsage` | 2 | usage error |
| `ExitUnsupported` | 3 | authorization-decline / domain-unsupported |
| `ExitInterrupted` | 130 | interrupted (`128 + SIGINT`) |

Validators: `IsSemverVersion`, `ValidateVersion`, `ValidateExitCode`.

## Behavioral conformance

The standalone runner is [`qyvora-conformance`](../qyvora-conformance):

```bash
cd ../qyvora-conformance
go run ./cmd/qyvora-conformance -root ../.. -json
```

It builds each framework, probes the real binary, checks version/exit-code
behavior, validates the capabilities document, and confirms the retired
per-tool TUI session is not linked.

## Capability validation

```bash
go run ./cmd/qyvora-capvalidator -root ../../frameworks
```

This checks capability coverage, output formats, tiers, noise levels and legal
files. It complements, but does not replace, behavioral conformance.

## Related contracts

- [`qyvora-models`](../../platform/qyvora-models) — Go wire types used by the
  platform, events and intelligence layers.
- [`qyvora-conformance`](../qyvora-conformance) — behavioral conformance.
- [`QYVORA-TOOL-OUTPUT-SPEC.md`](../../../knowledge/qyvora-docs/09-technical/cross-project/QYVORA-TOOL-OUTPUT-SPEC.md)
  — normative output contract.
