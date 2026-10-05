// Command qyvora-conformance validates each QYVORA framework binary against
// the minimum common machine contract for orchestrator/agent use. It builds
// each framework in the workspace, probes it through its own binary, and
// prints a pass/fail matrix.
//
// Usage:
//
//	qyvora-conformance [--root DIR] [--skip-build] [--bin PATH[:name]] [--json]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/QYVORA/qyvora-common/conformance"
)

func main() {
	root := flag.String("root", "", "workspace root (default: parent of this module)")
	skipBuild := flag.Bool("skip-build", false, "use existing built binaries instead of building")
	binSpec := flag.String("bin", "", "binary path(s) to test directly, e.g. /tmp/qf-mansa or /tmp/a.bin:mansa (comma-separated)")
	asJSON := flag.Bool("json", false, "emit results as JSON")
	flag.Parse()

	var fs []conformance.Framework
	if *binSpec != "" {
		for _, spec := range strings.Split(*binSpec, ",") {
			spec = strings.TrimSpace(spec)
			if spec == "" {
				continue
			}
			name := strings.TrimSuffix(filepath.Base(spec), filepath.Ext(spec))
			if i := strings.LastIndex(spec, ":"); i > 0 && strings.Count(spec, ":") == 1 {
				name = spec[i+1:]
				spec = spec[:i]
			}
			f, ok := conformance.Descriptor(name)
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown framework for %q\n", spec)
				os.Exit(1)
			}
			f.Bin = spec
			fs = append(fs, f)
		}
	} else {
		var err error
		ws := *root
		if ws == "" {
			self, _ := os.Executable()
			ws = selfDir(self)
		}
		fs, err = conformance.Workspace(ws)
		if err != nil {
			fmt.Fprintf(os.Stderr, "workspace: %v\n", err)
			os.Exit(1)
		}
		if len(fs) == 0 {
			fmt.Fprintf(os.Stderr, "no qyvora-* repos found under %s\n", ws)
			os.Exit(1)
		}
		if !*skipBuild {
			buildDir, _ := os.MkdirTemp("", "qf-build-")
			for i := range fs {
				if err := fs[i].Build(buildDir); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				}
			}
		}
	}

	sort.Slice(fs, func(i, j int) bool { return fs[i].Name < fs[j].Name })
	results := make([]conformance.Result, 0, len(fs))
	for _, f := range fs {
		results = append(results, f.Run())
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	} else {
		render(results)
	}
	for _, r := range results {
		if !r.Passed() {
			os.Exit(1)
		}
	}
}

func render(results []conformance.Result) {
	fmt.Printf("%-12s %-10s %-22s %-12s %-12s %-12s %-12s   %s\n", "framework", "version", "version-semver", "usage-exit-2", "unknown-cmd-2", "capabilities", "tui-bundled", "result")
	failed := 0
	for _, r := range results {
		byName := map[string]bool{}
		for _, c := range r.Checks {
			byName[c.Name] = c.Pass
		}
		row := fmt.Sprintf("%-12s %-10s %-22s %-12s %-12s %-12s %-12s   ",
			r.Framework, r.Version,
			mark(byName["version-semver"] && byName["version-exit0"]),
			mark(byName["usage-exit-2"]),
			mark(byName["unknown-cmd-exit-2"]),
			mark(byName["capabilities"]),
			mark(byName["tui-bundled"]),
		)
		if r.Passed() {
			fmt.Println(row + "PASS")
		} else {
			failed++
			fmt.Println(row + "FAIL")
			for _, c := range r.Checks {
				if !c.Pass {
					fmt.Printf("    %-22s %s\n", c.Name, c.Detail)
				}
			}
		}
	}
	fmt.Printf("\n%d/%d frameworks fully conformant\n", len(results)-failed, len(results))
}

func mark(b bool) string {
	if b {
		return "ok"
	}
	return "----"
}

func selfDir(self string) string {
	if dir := os.Getenv("QYVORA_WS"); dir != "" {
		return dir
	}
	cwd, _ := os.Getwd()
	return cwd
}
