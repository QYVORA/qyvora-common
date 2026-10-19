// Package conformance verifies that a real framework binary honours the
// minimum common machine contract. It shells out to the framework's own
// binary — never imports its code — so the checks exercise exactly what an
// orchestrator or agent would see.
package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/QYVORA/qyvora-common/contract"
)

// Framework describes how to build and probe one QYVORA framework.
type Framework struct {
	Name        string   // canonical name, e.g. "mansa"
	RepoDir     string   // absolute path to the repository root
	BuildPkg    string   // package to build, e.g. "./cmd/mansa" or "." for repo-root builds
	Package     string   // go module package path used by go build (e.g. "github.com/QYVORA/qyvora-mansa"). Empty = match lazily.
	Bin         string   // resolved binary path; empty until built
	VersionJSON []string // args producing machine-readable version, e.g. {"version","-o","json"}
	CapJSON     []string // args producing machine-readable capabilities, or nil if the framework has no capabilities command
	EventsFlag  string   // the events flag name, e.g. "--events", or "" if absent
	FormatFlag  string   // the output-format flag, e.g. "-o"
}

// TUIModulePath is the Go module the shared terminal UI lives in. The
// per-tool session was retired fleet-wide (frameworks fca91fc, "retire
// per-tool TUI in favor of one-shot CLI") in favour of the one-shot CLI, so
// the distribution check asserts the session engine (see tuiSessionSymbols)
// is NOT linked into a binary and flips back when the session returns with
// qyvora-tui's redirect to the AI orchestration layer (Phase 5).
//
// The module itself is allowed to be linked: its root package also carries
// the shared CLI brand helpers (RenderCLIArt, InTUISession, ThinBanner) that
// five tools render banners through, and the retirement commit kept that
// wiring on purpose. Module-level linkage therefore proves nothing about the
// session either way — the symbol table is what discriminates.
const TUIModulePath = "github.com/QYVORA/qyvora-tui"

// tuiSessionSymbols are the session engine's symbols in TUIModulePath: the
// entry point that starts the interactive session, the runner that executes
// commands inside it, and the cobra adapter that feeds it the command tree.
// Every framework references them only from its per-tool tui.go, all of which
// sit behind //go:build ignore, so any of these appearing in a binary means a
// framework re-registered the retired session. The shared CLI helpers are
// deliberately absent from this list: helper linkage is legal.
var tuiSessionSymbols = []string{
	TUIModulePath + ".Run",
	TUIModulePath + ".(*InProcessRunner)",
	TUIModulePath + ".CobraCommands",
}

// Check is a single conformance verdict.
type Check struct {
	Framework string `json:"framework"`
	Name      string `json:"check"`
	Pass      bool   `json:"pass"`
	Detail    string `json:"detail,omitempty"`
}

// Result is one framework's full conformance result.
type Result struct {
	Framework string  `json:"framework"`
	Version   string  `json:"version"`
	Bin       string  `json:"binary"`
	Checks    []Check `json:"checks"`
}

// Passed reports whether every check passed.
func (r Result) Passed() bool {
	for _, c := range r.Checks {
		if !c.Pass {
			return false
		}
	}
	return true
}

// Workspace discovers all qyvora-* repositories under root.
func Workspace(root string) ([]Framework, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var fs []Framework
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "qyvora-") || e.Name() == "qyvora-common" {
			continue
		}
		name := strings.TrimPrefix(e.Name(), "qyvora-")
		f, ok := known(name)
		if !ok {
			continue
		}
		f.RepoDir = filepath.Join(root, e.Name())
		f.BuildPkg = buildPkg(f.RepoDir, name)
		fs = append(fs, f)
	}
	return fs, nil
}

// buildPkg returns the package to build for a repo: ./cmd/<name> if it holds
// non-test Go files, else the repo root (".").
func buildPkg(repoDir, name string) string {
	pkg := filepath.Join("cmd", name)
	if entries, err := os.ReadDir(filepath.Join(repoDir, pkg)); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				return "./" + pkg
			}
		}
		for _, e := range entries {
			if e.IsDir() {
				if sub, err := os.ReadDir(filepath.Join(repoDir, pkg, e.Name())); err == nil {
					for _, s := range sub {
						if strings.HasSuffix(s.Name(), ".go") && !strings.HasSuffix(s.Name(), "_test.go") {
							return "./" + filepath.Join(pkg, e.Name())
						}
					}
				}
			}
		}
	}
	return "."
}

// Descriptor returns the probe descriptor for a known framework name.
func Descriptor(name string) (Framework, bool) { return known(name) }

// known returns the probe descriptor for a known framework.
func known(name string) (Framework, bool) {
	var f Framework
	switch name {
	case "aksum":
		f = Framework{Name: "aksum", VersionJSON: []string{"version", "-o", "json"}}
	case "anansi":
		f = Framework{Name: "anansi", VersionJSON: []string{"version", "-o", "json"}}
	case "toha3ee":
		f = Framework{Name: "toha3ee", VersionJSON: []string{"version", "-o", "json"}}
	case "jabari":
		f = Framework{Name: "jabari", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "nzinga":
		f = Framework{Name: "nzinga", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "mansa":
		f = Framework{Name: "mansa", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "shaka":
		f = Framework{Name: "shaka", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "sekhmet":
		f = Framework{Name: "sekhmet", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "imhotep", "timbuktu", "sundiata", "amanirenas", "kush":
		f = Framework{Name: name, VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "amina":
		// Amina distinguishes an unknown command from a target positional: a bare
		// invocation is an assessment, but an unrecognised word is a usage error,
		// so the unknown-cmd check applies.
		f = Framework{Name: "amina", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	default:
		return f, false
	}
	return f, true
}

// conformanceVersion is what the runner stamps into its own builds. It
// satisfies the contract's semver rule and is deliberately unlike any release
// tag, so a conformance report can never be mistaken for a shipped artifact.
const conformanceVersion = "9.9.9"

// Build compiles the framework binary into buildDir, stamped the way the
// release pipeline stamps it. The QYVORA output spec (section 4) says version
// strings are stamped at release time via -ldflags -X …internal/version.Version
// and that release artifacts MUST NOT report "dev" — while documenting the
// unstamped dev default as correct. So the runner exercises that exact wiring
// instead of asserting the default: a framework whose version variable moved
// or was renamed still links (the linker ignores -X for symbols it cannot
// find), reports "dev", and fails version-semver below, which is the breakage
// this stamping exists to surface — goreleaser would ship the same way.
func (f *Framework) Build(buildDir string) error {
	target := filepath.Join(buildDir, "qf-"+f.Name)
	if f.Package != "" {
		target = filepath.Join(buildDir, f.Package[strings.LastIndex(f.Package, "/")+1:])
	}
	args := []string{"build", "-o", target}
	if mod := f.modulePath(); mod != "" {
		args = append(args, "-ldflags", "-X "+mod+"/internal/version.Version="+conformanceVersion)
	}
	args = append(args, f.BuildPkg)
	cmd := exec.Command("go", args...)
	cmd.Dir = f.RepoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build %s: %w: %s", f.Name, err, strings.TrimSpace(string(out)))
	}
	f.Bin = target
	return nil
}

// modulePath reads the framework's module path from its go.mod. A miss leaves
// the build unstamped, and version-semver then reports the "dev" default —
// a truthful failure rather than a silently skipped stamp.
func (f *Framework) modulePath() string {
	data, err := os.ReadFile(filepath.Join(f.RepoDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if p, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(p)
		}
	}
	return ""
}

// Run executes all conformance checks against the built binary.
func (f Framework) Run() Result {
	res := Result{Framework: f.Name, Version: "", Bin: f.Bin, Checks: []Check{}}
	if f.Bin == "" {
		res.Checks = append(res.Checks, Check{f.Name, "binary", false, "not built"})
		return res
	}

	// 1. Identity: machine-readable version + semver, never dev.
	stdout, code, err := f.run(f.VersionJSON...)
	version := ""
	checks := []Check{}
	if err != nil {
		checks = append(checks, Check{f.Name, "version", false, fmt.Sprintf("run failed: %v", err)})
	} else {
		var vr contract.VersionResult
		if jsonErr := json.Unmarshal([]byte(stdout), &vr); jsonErr != nil {
			checks = append(checks, Check{f.Name, "version", false, fmt.Sprintf("version output not JSON: %v: %.60s", jsonErr, stdout)})
		} else {
			version = vr.Version
			err := contract.ValidateVersion(vr.Framework, vr.Version)
			ok := err == nil && code == contract.ExitSuccess && (vr.Framework == "" || vr.Framework == f.Name)
			detail := ""
			if err != nil {
				detail = err.Error()
			} else if code != contract.ExitSuccess {
				detail = fmt.Sprintf("exit %d", code)
			} else if vr.Framework != "" && vr.Framework != f.Name {
				detail = fmt.Sprintf("framework mismatch: %q", vr.Framework)
			} else if vr.Framework == "" {
				detail = "framework field absent"
				ok = false
			}
			checks = append(checks, Check{f.Name, "version-semver", ok, detail})
			if ok {
				checks = append(checks, Check{f.Name, "version-exit0", code == contract.ExitSuccess, fmt.Sprintf("exit %d", code)})
			}
		}
	}
	res.Version = version

	// 2. Exit contract: an unknown flag is usage (2), never runtime (1).
	_, code, err = f.run("--definitely-not-a-flag-xz")
	checks = append(checks, Check{f.Name, "usage-exit-2", err == nil && code == contract.ExitUsage,
		fmt.Sprintf("unknown flag exit=%d err=%v", code, err)})

	// 3. Exit contract: an unknown command is usage (2), where the CLI shape
	// lets us distinguish a command from a target positional. Every framework
	// now rejects a mistyped command; anansi's root takes a bare target, but its
	// Args validation classifies dotless words as unknown commands so the same
	// check applies.
	_, code, err = f.run("definitely-not-a-command-xz")
	checks = append(checks, Check{f.Name, "unknown-cmd-exit-2", err == nil && code == contract.ExitUsage,
		fmt.Sprintf("unknown command exit=%d err=%v", code, err)})

	// 4. Discovery: capabilities command, when advertised, must work.
	if f.CapJSON != nil {
		_, code, err = f.run(f.CapJSON...)
		checks = append(checks, Check{f.Name, "capabilities", err == nil && code == contract.ExitSuccess,
			fmt.Sprintf("capabilities %v exit=%d err=%v", f.CapJSON, code, err)})
	} else {
		// Skills-level note: no capabilities command is a known gap, not a failure.
		checks = append(checks, Check{f.Name, "capabilities", true, "no capabilities command (documented gap)"})
	}

	// 5. Distribution: the retired session must stay retired. A linked
	// session engine means a framework re-registered the retired command
	// against fca91fc; the shared banner helpers may remain linked.
	checks = append(checks, checkTUIRetired(f.Name, f.Bin))

	res.Checks = checks
	return res
}

// checkTUIRetired inspects bin's symbols and reports whether the retired
// session stayed retired.
//
// It inspects the binary rather than the repository's go.mod on purpose:
// go.mod records the requirement, only the binary records what was wired.
// The verdict rests on the session engine's symbols (tuiSessionSymbols)
// because the module's shared CLI helpers are linked by design in every tool
// that renders its banner through them — module linkage alone would fail the
// five tools that legitimately kept that wiring. go version -m is read too,
// but only so the detail can name the linked module's version.
func checkTUIRetired(name, bin string) Check {
	const checkName = "tui-retired"
	symOut, symErr := exec.Command("go", "tool", "nm", bin).CombinedOutput()
	if symErr != nil {
		detail := strings.TrimSpace(string(symOut))
		if detail == "" {
			detail = symErr.Error()
		}
		// Not being able to read the symbols is not evidence the session is
		// unlinked, but it is not evidence it is linked either, so it fails
		// rather than passing silently.
		return Check{name, checkName, false, "cannot inspect symbols: " + detail}
	}
	infoOut, _ := exec.Command("go", "version", "-m", bin).CombinedOutput()
	return tuiRetiredVerdict(name, string(infoOut), string(symOut))
}

// tuiRetiredVerdict applies the retirement rule to the binary's symbol table,
// with the build info supplying the module version for the message: a linked
// session engine fails (the session stays retired until Phase 5); a module
// linked only for its shared CLI banner helpers passes; a binary with no
// trace of the module passes.
func tuiRetiredVerdict(name, buildInfo, symbols string) Check {
	const checkName = "tui-retired"
	version, _ := tuiModuleVersion(buildInfo)
	ver := ""
	if version != "" {
		ver = " " + version
	}
	if linked := tuiSessionSymbolsLinked(symbols); len(linked) > 0 {
		short := make([]string, len(linked))
		for i, s := range linked {
			short[i] = strings.TrimPrefix(s, TUIModulePath+".")
		}
		return Check{name, checkName, false, fmt.Sprintf(
			"%s%s linked session engine (%s); the session stays retired until Phase 5",
			TUIModulePath, ver, strings.Join(short, ", "))}
	}
	if strings.Contains(symbols, TUIModulePath+".") {
		return Check{name, checkName, true, fmt.Sprintf(
			"session not linked; %s%s linked for shared CLI banner helpers only",
			TUIModulePath, ver)}
	}
	return Check{name, checkName, true, "session not linked"}
}

// tuiSessionSymbolsLinked returns which of tuiSessionSymbols appear in
// `go tool nm` output. Symbol lines are space-separated — address, kind,
// name — so the name is the last field. A name matches exactly (a
// package-level function such as ...qyvora-tui.Run) or with a dotted suffix
// (a method on InProcessRunner, whose symbol reads
// ...qyvora-tui.(*InProcessRunner).Run), while the dotted suffix keeps
// Run from matching an unrelated RunSomething.
func tuiSessionSymbolsLinked(symbols string) []string {
	var found []string
	seen := map[string]bool{}
	for _, line := range strings.Split(symbols, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		sym := fields[len(fields)-1]
		for _, want := range tuiSessionSymbols {
			if seen[want] || (sym != want && !strings.HasPrefix(sym, want+".")) {
				continue
			}
			seen[want] = true
			found = append(found, want)
		}
	}
	return found
}

// tuiModuleVersion extracts the TUI module's version from `go version -m`
// output and reports whether the build info claims its code was linked into
// the binary. The retirement verdict turns on the symbol table, not this
// flag; the flag still answers what the build info itself records, which is
// the difference between a requirement satisfied and code compiled in.
//
// A build-info line is tab-separated and the module path is not necessarily
// the first field: a dependency reads "dep", path, version, "h1:" hash. So the
// path is located by value and the version is the field after it, which holds
// whether the line is a dep, a replace source, or anything else.
//
// The h1: hash is the discriminator. A module that is required but whose
// packages were never imported is listed without one, so treating a line's
// mere presence as proof of linkage would report compiled-out code as wired.
//
// A module resolved through a replace directive is split across two lines,
// with the "=>" continuation carrying the hash. No framework replaces the TUI
// today, but reading only the dep line would report such a binary as
// unlinked, which is wrong rather than merely unhelpful.
func tuiModuleVersion(buildInfo string) (version string, linked bool) {
	lines := strings.Split(buildInfo, "\n")
	for i, line := range lines {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		at := -1
		for j, f := range fields {
			if strings.TrimSpace(f) == TUIModulePath {
				at = j
				break
			}
		}
		if at < 0 {
			continue
		}
		if at+1 < len(fields) {
			version = strings.TrimSpace(fields[at+1])
		}
		if hasContentHash(fields) {
			return version, true
		}
		// Replaced module: the hash moved to the "=>" continuation line.
		if i+1 < len(lines) {
			next := strings.Split(strings.TrimSpace(lines[i+1]), "\t")
			if len(next) > 0 && strings.TrimSpace(next[0]) == "=>" {
				if replaced := replacementVersion(next); replaced != "" {
					version = replaced
				}
				if hasContentHash(next) {
					return version, true
				}
				// A filesystem replacement (=> ../dir, => ./dir, => /abs/dir)
				// carries no h1 hash because the source is not in the module
				// cache, but a replace directive only satisfies the import, so
				// the TUI's code is necessarily compiled into the binary.
				if localPath := replacementPath(next); localPath {
					return version, true
				}
			}
		}
		return version, false
	}
	return "", false
}

// hasContentHash reports whether a build-info line carries an "h1:" hash,
// which the linker records only for modules compiled into the binary.
func hasContentHash(fields []string) bool {
	for _, f := range fields {
		if strings.HasPrefix(strings.TrimSpace(f), "h1:") {
			return true
		}
	}
	return false
}

// replacementVersion pulls the version out of an "=>" continuation line.
func replacementVersion(fields []string) string {
	for _, f := range fields[1:] {
		if v := strings.TrimSpace(f); strings.HasPrefix(v, "v") {
			return v
		}
	}
	return ""
}

// replacementPath reports whether the fields of an "=>" continuation line name
// a local directory replacement (.., ., or an absolute path). Such a module is
// built from source on disk, so its packages are compiled into the binary.
// A module-version replacement (a full module path) is not local and needs the
// h1: hash to prove linkage.
func replacementPath(fields []string) bool {
	for _, f := range fields[1:] {
		f = strings.TrimSpace(f)
		if f == "" || f == "=>" {
			continue
		}
		if strings.HasPrefix(f, "./") || strings.HasPrefix(f, "../") || strings.HasPrefix(f, "/") {
			return true
		}
	}
	return false
}

// run executes the binary with args under a timeout, returning combined
// stdout (capped) and the exit code. A non-zero exit still returns the
// captured output without error.
func (f Framework) run(args ...string) (string, int, error) {
	ctx, cancel := (&runCtx{timeout: 15 * time.Second}).ctx()
	defer cancel()
	var stdout cappedBuffer
	var stderr cappedBuffer
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			return "", -1, err
		}
	}
	_ = stderr
	return stdout.String(), code, nil
}

const maxProbeOutput = 256 << 10

// cappedBuffer keeps at most maxProbeOutput bytes of output.
type cappedBuffer struct {
	buf bytes.Buffer
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.buf.Len() < maxProbeOutput {
		remain := maxProbeOutput - c.buf.Len()
		if len(p) > remain {
			p = p[:remain]
		}
		c.buf.Write(p)
	}
	return len(p), nil
}

func (c *cappedBuffer) String() string { return c.buf.String() }

// runCtx threads a timeout through the probe helpers.
type runCtx struct{ timeout time.Duration }

func (r *runCtx) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.timeout)
}
