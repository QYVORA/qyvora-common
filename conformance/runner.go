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
	"regexp"
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
	// SkipUnknownCmd marks frameworks whose root command accepts an arbitrary
	// positional as a target (so "bogus-xyz" cannot be distinguished from a
	// valid invocation). The unknown-cmd exit-code check is skipped for them.
	SkipUnknownCmd bool
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

var semverSuffixRE = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+`)

// Descriptor returns the probe descriptor for a known framework name.
func Descriptor(name string) (Framework, bool) { return known(name) }

// known returns the probe descriptor for a known framework.
func known(name string) (Framework, bool) {
	var f Framework
	switch name {
	case "aksum":
		f = Framework{Name: "aksum", VersionJSON: []string{"version", "--format", "json"}}
	case "anansi":
		f = Framework{Name: "anansi", VersionJSON: []string{"version", "-o", "json"}, SkipUnknownCmd: true}
	case "toha3ee":
		f = Framework{Name: "toha3ee", VersionJSON: []string{"version", "-o", "json"}}
	case "jabari":
		f = Framework{Name: "jabari", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "nzinga":
		f = Framework{Name: "nzinga", VersionJSON: []string{"version", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "mansa":
		f = Framework{Name: "mansa", VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "shaka":
		f = Framework{Name: "shaka", VersionJSON: []string{"version", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "sekhmet":
		f = Framework{Name: "sekhmet", VersionJSON: []string{"version", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	case "imhotep", "timbuktu", "sundiata", "amanirenas", "kush": // this line seems to be wrong compared to other lines at the top for the other frameworks
		f = Framework{Name: name, VersionJSON: []string{"version", "-o", "json"}, CapJSON: []string{"capabilities", "-o", "json"}, EventsFlag: "--events", FormatFlag: "-o"}
	default:
		return f, false
	}
	return f, true
}

// Build compiles the framework binary into buildDir.
func (f *Framework) Build(buildDir string) error {
	target := filepath.Join(buildDir, "qf-"+f.Name)
	if f.Package != "" {
		target = filepath.Join(buildDir, f.Package[strings.LastIndex(f.Package, "/")+1:])
	}
	cmd := exec.Command("go", "build", "-o", target, f.BuildPkg)
	cmd.Dir = f.RepoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build %s: %w: %s", f.Name, err, strings.TrimSpace(string(out)))
	}
	f.Bin = target
	return nil
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
	// lets us distinguish a command from a target positional.
	if !f.SkipUnknownCmd {
		_, code, err = f.run("definitely-not-a-command-xz")
		checks = append(checks, Check{f.Name, "unknown-cmd-exit-2", err == nil && code == contract.ExitUsage,
			fmt.Sprintf("unknown command exit=%d err=%v", code, err)})
	}

	// 4. Discovery: capabilities command, when advertised, must work.
	if f.CapJSON != nil {
		_, code, err = f.run(f.CapJSON...)
		checks = append(checks, Check{f.Name, "capabilities", err == nil && code == contract.ExitSuccess,
			fmt.Sprintf("capabilities %v exit=%d err=%v", f.CapJSON, code, err)})
	} else {
		// Skills-level note: no capabilities command is a known gap, not a failure.
		checks = append(checks, Check{f.Name, "capabilities", true, "no capabilities command (documented gap)"})
	}

	res.Checks = checks
	return res
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
