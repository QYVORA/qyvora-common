package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Build info lines are tab-separated; the fixtures below are shaped from real
// `go version -m` output rather than invented, because the version lookup
// turns on field position and the h1: marker. Symbol fixtures are shaped from
// real `go tool nm` output for the same reason: the retirement verdict reads
// the last field of each line.

func TestTUIModuleVersionLinked(t *testing.T) {
	// A dep line carrying an h1: hash is the only shape that proves the
	// module's packages were compiled into the binary.
	info := "aksum: go1.26.8\n" +
		"\tpath\tgithub.com/QYVORA/qyvora-aksum\n" +
		"\tdep\tgithub.com/QYVORA/qyvora-tui\tv0.7.1\th1:uTxgdCgJAfPZ48MWvpEePuIUT/TFeT/CMKJZxtzfyM4=\n" +
		"\tdep\tgithub.com/atotto/clipboard\tv0.1.4\th1:EH0zSVneZPSuFR11BlR9YppQTVDbh5+16AmcJi4g1z4=\n"
	version, linked := tuiModuleVersion(info)
	if !linked {
		t.Fatal("a dep line with an h1: hash must report linked")
	}
	if version != "v0.7.1" {
		t.Errorf("version = %q, want v0.7.1", version)
	}
}

// The normal post-retirement state: go.mod requires the module, `go mod tidy`
// is satisfied, nothing imports it, and the binary ships as the one-shot CLI.
// The check must pass this, not fail it.
func TestTUIModuleVersionRequiredButNotLinked(t *testing.T) {
	info := "aksum: go1.26.8\n" +
		"\tpath\tgithub.com/QYVORA/qyvora-aksum\n" +
		"\tdep\t" + TUIModulePath + "\tv0.7.1\t\n"
	version, linked := tuiModuleVersion(info)
	if linked {
		t.Error("a dep line without an h1: hash must not report linked")
	}
	if version != "v0.7.1" {
		t.Errorf("version = %q, want v0.7.1 so the failure names what is missing", version)
	}
}

func TestTUIModuleVersionAbsent(t *testing.T) {
	info := "aksum: go1.26.8\n" +
		"\tpath\tgithub.com/QYVORA/qyvora-aksum\n" +
		"\tdep\tgithub.com/atotto/clipboard\tv0.1.4\th1:EH0zSVneZPSuFR11BlR9YppQTVDbh5+16AmcJi4g1z4=\n"
	version, linked := tuiModuleVersion(info)
	if linked || version != "" {
		t.Errorf("got version=%q linked=%v, want absent", version, linked)
	}
}

// A module resolved through a replace directive splits across two lines, with
// the hash on the "=>" continuation. Reading only the dep line would call such
// a binary unlinked, which is a false failure rather than a useful signal.
func TestTUIModuleVersionReplaced(t *testing.T) {
	info := "aksum: go1.26.8\n" +
		"\tdep\t" + TUIModulePath + "\tv0.7.1\t\n" +
		"\t=>\tgithub.com/fork/qyvora-tui\tv0.8.0\th1:uTxgdCgJAfPZ48MWvpEePuIUT/TFeT/CMKJZxtzfyM4=\n"
	version, linked := tuiModuleVersion(info)
	if !linked {
		t.Fatal("a replaced module with an h1: on the => line must report linked")
	}
	if version != "v0.8.0" {
		t.Errorf("version = %q, want the replacement version v0.8.0", version)
	}
}

// A replaced module with no hash anywhere is still not linked.
func TestTUIModuleVersionReplacedWithoutHash(t *testing.T) {
	info := "\tdep\t" + TUIModulePath + "\tv0.7.1\t\n" +
		"\t=>\tgithub.com/fork/qyvora-tui\tv0.8.0\t\n"
	if _, linked := tuiModuleVersion(info); linked {
		t.Error("a => line without an h1: hash must not report linked")
	}
}

// A filesystem replacement has no h1: hash, but a local directory is the build
// source itself, so the TUI code is necessarily compiled into the binary.
func TestTUIModuleVersionReplacedByLocalDir(t *testing.T) {
	info := "\tdep\t" + TUIModulePath + "\tv0.9.0\t\n" +
		"\t=>\t../qyvora-tui\t\n"
	if _, linked := tuiModuleVersion(info); !linked {
		t.Error("a => line naming a local directory must report linked")
	}
}

// A stdlib-only binary has no TUI, which is exactly what the retirement
// contract wants: the probe reads its build info and the verdict passes. The
// paired fixture test below proves the verdict still fails when a TUI is
// linked, so the check discriminates rather than passing everything.
func TestCheckTUIRetiredPassesForBinaryWithoutTUI(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}
	dir := t.TempDir()
	// A module file is required even for a stdlib-only program: without one
	// `go build` refuses to run in module mode.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "notui")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v: %s", err, out)
	}

	got := checkTUIRetired("fixture", bin)
	if !got.Pass {
		t.Fatalf("a binary with no TUI failed the retirement check: %+v", got)
	}
}

// The failure the check exists for: session engine symbols surviving in the
// binary mean a framework re-registered the retired session against the
// Phase-5 decision. Symbol lines are shaped from real `go tool nm` output —
// address, kind, name — where the helper-only binaries carry none of these.
func TestTUIRetiredVerdictFailsWhenLinked(t *testing.T) {
	info := "jabari: go1.26.8\n" +
		"\tpath\tgithub.com/QYVORA/qyvora-jabari\n" +
		"\tdep\t" + TUIModulePath + "\tv0.7.1\th1:uTxgdCgJAfPZ48MWvpEePuIUT/TFeT/CMKJZxtzfyM4=\n"
	symbols := "  851b20 T " + TUIModulePath + ".Run\n" +
		"  851c80 T " + TUIModulePath + ".(*InProcessRunner).Run\n" +
		"  851d40 T " + TUIModulePath + ".CobraCommands\n"
	got := tuiRetiredVerdict("fixture", info, symbols)
	if got.Pass {
		t.Fatalf("a linked session engine passed the retirement check: %+v", got)
	}
	if !strings.Contains(got.Detail, TUIModulePath) {
		t.Errorf("failure detail does not name the linked module: %q", got.Detail)
	}
	if !strings.Contains(got.Detail, "Run") || !strings.Contains(got.Detail, "(*InProcessRunner)") {
		t.Errorf("failure detail does not name the linked engine symbols: %q", got.Detail)
	}
}

// Helper linkage is legal: five tools render their brand banner through the
// shared CLI helpers (RenderCLIArt, InTUISession), which the retirement
// commit kept wired on purpose. Real `go tool nm` output from jabari is
// entirely of this shape, and the check must pass it while still failing the
// linked engine above — that contrast is what makes the check discriminate.
func TestTUIRetiredVerdictPassesForHelpersOnly(t *testing.T) {
	info := "jabari: go1.26.8\n" +
		"\tpath\tgithub.com/QYVORA/qyvora-jabari\n" +
		"\tdep\t" + TUIModulePath + "\tv0.11.0\th1:kk4XNTRSlPUd1p2sJ7R1FeEIDOEYzmJwZ2oTK4pfoCI=\n"
	symbols := "  e52030 D " + TUIModulePath + "..inittask\n" +
		"  851b20 T " + TUIModulePath + ".RenderCLIArt\n" +
		"  851220 T " + TUIModulePath + ".init.func1\n"
	got := tuiRetiredVerdict("fixture", info, symbols)
	if !got.Pass {
		t.Fatalf("helper-only linkage failed the retirement check: %+v", got)
	}
	if !strings.Contains(got.Detail, "helpers") {
		t.Errorf("pass detail does not say the linkage is helpers-only: %q", got.Detail)
	}
}

// An unreadable or non-Go binary must fail rather than pass silently: not
// being able to prove the TUI is unlinked is not evidence that it is.
func TestCheckTUIRetiredFailsOnUnreadableBinary(t *testing.T) {
	got := checkTUIRetired("missing", filepath.Join(t.TempDir(), "does-not-exist"))
	if got.Pass {
		t.Fatalf("a missing binary passed: %+v", got)
	}
	if got.Detail == "" {
		t.Error("failure carries no detail")
	}
}
