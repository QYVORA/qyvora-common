package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const version = "0.1.0"

// Framework represents one QYVORA security framework
type Framework struct {
	Name        string
	Description string
	BinaryPath  string
	Tier3Type   string   // "Shape A" (live exploitation) or "Shape B" (bounded proof)
	ExtraFlags  []string // Extra flags needed for all commands (e.g. --no-sudo for TOHA3EE)
}

// Check result represents the outcome of a single check
type CheckResult struct {
	Framework string
	Check     string
	Passed    bool
	Message   string
}

var frameworks = []Framework{
	{Name: "anansi", Description: "Web attack surface intelligence", Tier3Type: "Shape A"},
	{Name: "toha3ee", Description: "Network security assessment", Tier3Type: "Shape A", ExtraFlags: []string{"--no-sudo"}},
	{Name: "jabari", Description: "Android security assessment", Tier3Type: "Shape A"},
	{Name: "aksum", Description: "Binary analysis & reverse engineering", Tier3Type: "Shape B"},
	{Name: "nzinga", Description: "OSINT / intelligence", Tier3Type: "Shape B"},
	{Name: "shaka", Description: "Active Directory / Windows", Tier3Type: "Shape A"},
	{Name: "sekhmet", Description: "Fuzzing / vulnerability discovery", Tier3Type: "Shape B"},
	{Name: "mansa", Description: "Wireless (WLAN/BLE)", Tier3Type: "Shape A"},
	{Name: "amanirenas", Description: "iOS app security (offline)", Tier3Type: "Shape B"},
	{Name: "sundiata", Description: "Identity & access (AD directory files, offline)", Tier3Type: "Shape B"},
	{Name: "timbuktu", Description: "Incident response / digital forensics (offline)", Tier3Type: "Shape B"},
	{Name: "kush", Description: "Malware sample analysis (offline, never executes)", Tier3Type: "Shape B (static-only)"},
	{Name: "imhotep", Description: "Cloud snapshot analysis (offline)", Tier3Type: "Shape B"},
	{Name: "amina", Description: "Operational security / host exposure", Tier3Type: "Shape B"},
}

var (
	flagTool    = flag.String("tool", "", "check only this framework")
	flagReport  = flag.Bool("report", false, "generate gap report format")
	flagVerbose = flag.Bool("v", false, "verbose output")
	rootDir     string
)

func main() {
	flag.Usage = usage
	flag.Parse()

	// Determine root directory (parent of conformance tool)
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "conformance: cannot determine executable path: %v\n", err)
		os.Exit(2)
	}
	rootDir = filepath.Dir(filepath.Dir(exe))

	// Filter frameworks if --tool specified
	toCheck := frameworks
	if *flagTool != "" {
		found := false
		for _, fw := range frameworks {
			if fw.Name == *flagTool {
				toCheck = []Framework{fw}
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "conformance: unknown tool %q\n", *flagTool)
			os.Exit(2)
		}
	}

	if !*flagReport {
		fmt.Printf("qyvora-conformance %s\n", version)
		fmt.Printf("Checking %d framework(s)...\n\n", len(toCheck))
	}

	allResults := []CheckResult{}
	totalChecks := 0
	passedChecks := 0

	for _, fw := range toCheck {
		results := checkFramework(fw)
		allResults = append(allResults, results...)
		for _, r := range results {
			totalChecks++
			if r.Passed {
				passedChecks++
			}
		}
	}

	if *flagReport {
		printGapReport(allResults)
	} else {
		printResults(allResults, totalChecks, passedChecks)
	}

	if passedChecks < totalChecks {
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `qyvora-conformance - Conformance testing for QYVORA frameworks

Usage:
  conformance [flags]

Flags:
  --tool <name>    Check only the specified framework
  --report         Generate gap report format (for PROMPT.md Phase 0)
  -v               Verbose output
  -h, --help       Show this help

Examples:
  conformance                    # Check all 14 frameworks
  conformance --tool anansi      # Check only anansi
  conformance --report           # Generate gap report

Exit codes:
  0  All checks passed
  1  One or more checks failed
  2  Usage error
`)
	os.Exit(2)
}

func checkFramework(fw Framework) []CheckResult {
	results := []CheckResult{}

	if *flagVerbose && !*flagReport {
		fmt.Printf("=== %s (%s) ===\n", fw.Name, fw.Description)
	}

	// Discover binary path
	binaryPath := findBinary(fw.Name)
	if binaryPath == "" {
		results = append(results, CheckResult{
			Framework: fw.Name,
			Check:     "binary_exists",
			Passed:    false,
			Message:   "binary not found",
		})
		if *flagVerbose && !*flagReport {
			fmt.Printf("  ✗ binary not found\n\n")
		}
		return results
	}
	fw.BinaryPath = binaryPath

	// Check: Binary exists and is executable
	results = append(results, CheckResult{
		Framework: fw.Name,
		Check:     "binary_exists",
		Passed:    true,
		Message:   fmt.Sprintf("found at %s", binaryPath),
	})

	// Check: Legal files exist
	for _, file := range []string{"LICENSE", "NOTICE", "SECURITY.md"} {
		path := filepath.Join(filepath.Dir(binaryPath), "..", file)
		exists := fileExists(path)
		results = append(results, CheckResult{
			Framework: fw.Name,
			Check:     fmt.Sprintf("file_%s", strings.ToLower(strings.ReplaceAll(file, ".", "_"))),
			Passed:    exists,
			Message:   file,
		})
	}

	// Check: README exists
	readmePath := filepath.Join(filepath.Dir(binaryPath), "..", "README.md")
	exists := fileExists(readmePath)
	results = append(results, CheckResult{
		Framework: fw.Name,
		Check:     "file_readme",
		Passed:    exists,
		Message:   "README.md",
	})

	// Check: Version command works
	versionArgs := make([]string, len(fw.ExtraFlags))
	copy(versionArgs, fw.ExtraFlags)
	versionArgs = append(versionArgs, "version")
	versionWorks := runCommand(binaryPath, versionArgs, nil)
	results = append(results, CheckResult{
		Framework: fw.Name,
		Check:     "version_command",
		Passed:    versionWorks,
		Message:   "version command",
	})

	// Check: Capabilities command exists and outputs JSON
	// Try with -o json first (standard), then without (some tools default to JSON)
	capArgs := make([]string, len(fw.ExtraFlags))
	copy(capArgs, fw.ExtraFlags)
	capArgs = append(capArgs, "capabilities", "-o", "json")
	capJSON, capWorks := runCommandOutput(binaryPath, capArgs, nil)
	if !capWorks {
		// Try without -o flag (TOHA3EE style)
		capArgs = make([]string, len(fw.ExtraFlags))
		copy(capArgs, fw.ExtraFlags)
		capArgs = append(capArgs, "capabilities")
		capJSON, capWorks = runCommandOutput(binaryPath, capArgs, nil)
	}
	if !capWorks {
		// Try with --json flag (alternate style)
		capArgs = make([]string, len(fw.ExtraFlags))
		copy(capArgs, fw.ExtraFlags)
		capArgs = append(capArgs, "capabilities", "--json")
		capJSON, capWorks = runCommandOutput(binaryPath, capArgs, nil)
	}
	results = append(results, CheckResult{
		Framework: fw.Name,
		Check:     "capabilities_command",
		Passed:    capWorks,
		Message:   "capabilities command",
	})

	if capWorks && capJSON != "" {
		// Validate JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(capJSON), &data); err == nil {
			results = append(results, CheckResult{
				Framework: fw.Name,
				Check:     "capabilities_json_valid",
				Passed:    true,
				Message:   "capabilities JSON is valid",
			})
		} else {
			results = append(results, CheckResult{
				Framework: fw.Name,
				Check:     "capabilities_json_valid",
				Passed:    false,
				Message:   fmt.Sprintf("capabilities JSON invalid: %v", err),
			})
		}
	}

	// Check: Help command works
	helpWorks := runCommand(binaryPath, []string{"--help"}, nil)
	results = append(results, CheckResult{
		Framework: fw.Name,
		Check:     "help_command",
		Passed:    helpWorks,
		Message:   "help command",
	})

	if *flagVerbose && !*flagReport {
		for _, r := range results[1:] { // Skip first (binary_exists) since we already showed it
			if r.Passed {
				fmt.Printf("  ✓ %s\n", r.Message)
			} else {
				fmt.Printf("  ✗ %s\n", r.Message)
			}
		}
		fmt.Println()
	}

	return results
}

func findBinary(name string) string {
	// Try in ../qyvora-<name>/bin/<name>
	binPath := filepath.Join(rootDir, "qyvora-"+name, "bin", name)
	if fileExists(binPath) {
		return binPath
	}

	// Try in ../qyvora-<name>/<name>
	binPath = filepath.Join(rootDir, "qyvora-"+name, name)
	if fileExists(binPath) {
		return binPath
	}

	// Try in PATH
	path, err := exec.LookPath(name)
	if err == nil {
		return path
	}

	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runCommand(binary string, args []string, env []string) bool {
	cmd := exec.Command(binary, args...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	err := cmd.Run()
	return err == nil
}

func runCommandOutput(binary string, args []string, env []string) (string, bool) {
	cmd := exec.Command(binary, args...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(output), true
}

func printResults(results []CheckResult, total, passed int) {
	// Group by framework
	byFramework := make(map[string][]CheckResult)
	for _, r := range results {
		byFramework[r.Framework] = append(byFramework[r.Framework], r)
	}

	failedFrameworks := []string{}
	for _, fw := range frameworks {
		fwResults := byFramework[fw.Name]
		if len(fwResults) == 0 {
			continue
		}

		failed := 0
		for _, r := range fwResults {
			if !r.Passed {
				failed++
			}
		}

		status := "✓"
		if failed > 0 {
			status = "✗"
			failedFrameworks = append(failedFrameworks, fw.Name)
		}

		if !*flagVerbose {
			fmt.Printf("%s %s - %d/%d checks passed\n", status, fw.Name, len(fwResults)-failed, len(fwResults))
		}
	}

	fmt.Printf("\n")
	fmt.Printf("Summary: %d/%d checks passed across %d framework(s)\n", passed, total, len(byFramework))

	if len(failedFrameworks) > 0 {
		fmt.Printf("\nFailed frameworks: %s\n", strings.Join(failedFrameworks, ", "))
		fmt.Printf("\nRun with -v for details, or --report for gap analysis\n")
	} else {
		fmt.Printf("\n✓ All frameworks passed conformance checks\n")
	}
}

func printGapReport(results []CheckResult) {
	fmt.Println("# QYVORA Capability Gap Report")
	fmt.Println()
	fmt.Printf("Generated: %s\n", "2026-10-07") // TODO: use actual date
	fmt.Println("Tool: qyvora-conformance " + version)
	fmt.Println()
	fmt.Println("This report audits all 14 QYVORA frameworks against the standards defined in PROMPT.md.")
	fmt.Println()

	// Group by framework
	byFramework := make(map[string][]CheckResult)
	for _, r := range results {
		byFramework[r.Framework] = append(byFramework[r.Framework], r)
	}

	for _, fw := range frameworks {
		fwResults := byFramework[fw.Name]
		if len(fwResults) == 0 {
			continue
		}

		fmt.Printf("## %s\n\n", fw.Name)
		fmt.Printf("**Description:** %s  \n", fw.Description)
		fmt.Printf("**Tier 3 Type:** %s  \n\n", fw.Tier3Type)

		passed := 0
		failed := 0
		for _, r := range fwResults {
			if r.Passed {
				passed++
			} else {
				failed++
			}
		}

		fmt.Printf("**Status:** %d/%d checks passed\n\n", passed, len(fwResults))

		if failed > 0 {
			fmt.Println("**Gaps:**")
			for _, r := range fwResults {
				if !r.Passed {
					fmt.Printf("- ✗ %s: %s\n", r.Check, r.Message)
				}
			}
			fmt.Println()
		} else {
			fmt.Println("✓ No gaps found")
		}
	}

	fmt.Println("---")
	fmt.Println("## Summary")
	fmt.Println()

	totalPassed := 0
	totalChecks := 0
	for _, r := range results {
		totalChecks++
		if r.Passed {
			totalPassed++
		}
	}

	fmt.Printf("- **Total checks:** %d\n", totalChecks)
	fmt.Printf("- **Passed:** %d\n", totalPassed)
	fmt.Printf("- **Failed:** %d\n", totalChecks-totalPassed)
	fmt.Println()

	if totalPassed < totalChecks {
		fmt.Println("### Next Steps")
		fmt.Println()
		fmt.Println("Address gaps in priority order per PROMPT.md §5 phased build plan.")
	}
}
