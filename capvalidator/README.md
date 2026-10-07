# qyvora-conformance

**Capability coverage validator** for all 14 QYVORA security frameworks.

## Purpose

Validates that each QYVORA framework meets the **capability requirements** defined in `../PROMPT.md`:

- ✓ Capabilities command exists and outputs correct formats
- ✓ Output format support (terminal, json, markdown, yaml, html)
- ✓ Capability tiers (Tier 1/2/3) are declared
- ✓ Noise levels (passive/low/moderate/aggressive) are declared
- ✓ Legal files present (LICENSE, NOTICE, SECURITY.md)
- ✓ Help command registry has no duplicates
- ✓ All commands have descriptions
- ✓ Documentation matches actual capabilities

## Relationship to qyvora-common/conformance

**Two conformance tools exist** serving different purposes:

| Tool | Purpose | What it validates |
|------|---------|------------------|
| **qyvora-conformance** (this tool) | Phase 1 capability compliance | Output formats, tiers, noise levels, help registry, legal files |
| **qyvora-common/cmd/qyvora-conformance** | Phase 5 machine contract | Exit codes, TUI linkage, JSONL events, version output |

**Both are needed:**
- This tool validates **PROMPT.md requirements** for Phase 1 (framework standardization)
- qyvora-common validates **machine contract** for Phase 5 (AI orchestration readiness)

See `../CONFORMANCE-TOOLS-ANALYSIS.md` for detailed comparison.

## Usage

```bash
# Check all 14 frameworks
./conformance

# Check specific framework
./conformance --tool anansi

# Generate gap report
./conformance --report > QYVORA-CAPABILITY-GAP-REPORT.md

# Verbose output
./conformance -v
```

## Exit Codes

- `0` - All checks passed
- `1` - One or more checks failed
- `2` - Usage error

## Frameworks Checked

1. anansi - Web attack surface intelligence
2. toha3ee - Network security assessment
3. jabari - Android security assessment
4. aksum - Binary analysis & reverse engineering
5. nzinga - OSINT / intelligence
6. shaka - Active Directory / Windows
7. sekhmet - Fuzzing / vulnerability discovery
8. mansa - Wireless (WLAN/BLE)
9. amanirenas - iOS app security (offline)
10. sundiata - Identity & access (AD directory files, offline)
11. timbuktu - Incident response / digital forensics (offline)
12. kush - Malware sample analysis (offline, never executes)
13. imhotep - Cloud snapshot analysis (offline)
14. amina - Operational security / host exposure

## Checks Performed

### Basic Checks
- Binary builds successfully
- Legal files exist (LICENSE, NOTICE, SECURITY.md)
- README.md exists
- Version command works

### Capability Checks
- `capabilities` command exists
- Capabilities output is valid JSON
- Capabilities list tiers (Tier 1/2/3)
- Capabilities list noise levels

### Output Format Checks
- Terminal output works
- JSON output works and is valid
- Markdown output works
- YAML output works (where applicable)
- HTML output works (where applicable)

### Help Registry Checks  
- Help command works
- No duplicate commands
- All commands have Short description
- All commands have Usage string
- Flags have descriptions

## Implementation

Written in Go. Runs as a standalone binary that discovers framework binaries in `../<tool>/` or expects them in PATH.

