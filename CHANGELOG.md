# Changelog

All notable changes to `qyvora-common` are documented here.

## v0.2.1 - 2026-10-31

### Added

- `test/run-all-tests.sh` with fast and full modes.
- Full mode exercises `qyvora-capvalidator` against fake framework binaries,
  including valid/invalid capability JSON, missing binaries, unknown tools, and
  gap-report output.
- CI now runs the local suite.

## v0.2.0 - 2026-10-31

### Changed

- `contract.Envelope` now mirrors `qyvora-models.Event` with `event_type`,
  `tier`, `module`, `action`, `severity`, `target`, `session_id` and `run_id`.
- Behavioral conformance moved to the standalone `qyvora-conformance` module.

### Added

- CI workflow for formatting, vet, tests and build.
- LICENSE and release metadata.

### Removed

- Checked-in `qyvora-capvalidator` binaries.
- The in-module `conformance` package and `cmd/qyvora-conformance`; use
  `qyvora-conformance` instead.

## v0.1.0 - 2026-10-07

- Initial contract and capability validator.
