// Package contract defines the minimum common machine contract shared by the
// QYVORA frameworks, as proposed in the machine-layer readiness audit
// (docs/QYVORA-FRAMEWORK-MACHINE-LAYER-READINESS-AUDIT.md, section 14).
//
// The frameworks do NOT import this module: each implements the contract
// natively. This package is the reference schema used by the conformance
// runner to verify that a real binary honours the contract
// (subpackage qyvora-common/conformance).
package contract

import (
	"fmt"
	"regexp"
)

// Standard exit codes a framework must honour. Authorization-decline and
// domain-unsupported by convention use 3.
const (
	ExitSuccess     = 0
	ExitRuntime     = 1
	ExitUsage       = 2
	ExitUnsupported = 3
	ExitInterrupted = 130
)

// Identity is the identity block every machine stream and result carries.
type Identity struct {
	Framework     string `json:"framework"`
	Version       string `json:"version"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

// Envelope is the JSONL event envelope emitted on the event stream.
type Envelope struct {
	SchemaVersion string         `json:"schema_version"`
	Timestamp     string         `json:"timestamp"`
	ExecutionID   string         `json:"execution_id"`
	Framework     string         `json:"framework"`
	Level         string         `json:"level"`
	Event         string         `json:"event"`
	Data          map[string]any `json:"data,omitempty"`
}

// Context ties one run to a result, a target and an optional session.
type Context struct {
	ExecutionID string  `json:"execution_id"`
	SessionID   string  `json:"session_id,omitempty"`
	Target      *Target `json:"target,omitempty"`
}

// Target is the object an assessment runs against (type/value/authorization).
type Target struct {
	Type          string `json:"type"`
	Value         string `json:"value"`
	Authorization string `json:"authorization,omitempty"`
}

// Command is one capability record an orchestrator can call without guessing.
type Command struct {
	Name          string   `json:"name"`
	Inputs        []string `json:"inputs,omitempty"`
	Outputs       []string `json:"outputs,omitempty"`
	Dangerous     bool     `json:"dangerous"`
	Authorization string   `json:"authorization,omitempty"` // required | dry-run-only | none
	Emits         []string `json:"emits,omitempty"`
	Deterministic bool     `json:"deterministic"`
}

// CapabilityManifest is the machine-readable capability document.
type CapabilityManifest struct {
	Framework string    `json:"framework"`
	Version   string    `json:"version"`
	Commands  []Command `json:"commands"`
}

// VersionResult is the machine-readable version document.
type VersionResult struct {
	Framework string `json:"framework"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Date      string `json:"date,omitempty"`
	OS        string `json:"os,omitempty"`
	Arch      string `json:"arch,omitempty"`
}

// FindFinding is the per-finding record shape of a canonical result.
type FindFinding struct {
	ID          string   `json:"id,omitempty"`
	Rule        string   `json:"rule"`
	Title       string   `json:"title,omitempty"`
	Severity    string   `json:"severity"`
	Confidence  string   `json:"confidence,omitempty"`
	Target      string   `json:"target,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Timestamp   string   `json:"timestamp,omitempty"`
}

// Result is the canonical single JSON document for an assessment run.
type Result struct {
	Identity Identity      `json:"identity"`
	Context  Context       `json:"context"`
	Findings []FindFinding `json:"findings,omitempty"`
	Evidence []any         `json:"evidence,omitempty"`
	Risk     *Risk         `json:"risk,omitempty"`
	Events   []string      `json:"events,omitempty"`
}

// Risk is the target-level risk block.
type Risk struct {
	Score     int    `json:"score"`
	Level     string `json:"level"`
	Rationale string `json:"rationale,omitempty"`
}

var semverRE = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)

// IsSemverVersion reports whether v looks like a semantic version.
func IsSemverVersion(v string) bool {
	return v != "" && semverRE.MatchString(v)
}

// ValidateVersion enforces the identity rule: a shipped binary must report a
// semantic version, never a bare dev marker.
func ValidateVersion(framework, version string) error {
	if framework == "" {
		return fmt.Errorf("version document missing framework name")
	}
	if !IsSemverVersion(version) {
		return fmt.Errorf("framework %q reports version %q: must be semver, never 'dev'", framework, version)
	}
	return nil
}

// ValidateExitCode enforces the usage-vs-runtime distinction of the contract.
func ValidateExitCode(code int, want int) error {
	if code != want {
		return fmt.Errorf("expected exit code %d, got %d", want, code)
	}
	return nil
}
