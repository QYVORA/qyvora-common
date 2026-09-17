package contract

import "testing"

func TestIsSemverVersion(t *testing.T) {
	if !IsSemverVersion("0.1.0") {
		t.Error("0.1.0 must be valid")
	}
	if !IsSemverVersion("v0.1.0") {
		t.Error("v0.1.0 must be valid")
	}
	if IsSemverVersion("dev") {
		t.Error("dev must not be valid")
	}
	if IsSemverVersion("") {
		t.Error("empty must not be valid")
	}
}

func TestValidateVersion(t *testing.T) {
	if err := ValidateVersion("mansa", "0.1.0"); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}
	if err := ValidateVersion("mansa", "dev"); err == nil {
		t.Fatal("expected dev to be rejected")
	}
	if err := ValidateVersion("", "0.1.0"); err == nil {
		t.Fatal("expected empty framework to be rejected")
	}
}

func TestValidateExitCode(t *testing.T) {
	if err := ValidateExitCode(2, ExitUsage); err != nil {
		t.Fatalf("2=usage should pass: %v", err)
	}
	if err := ValidateExitCode(0, ExitUsage); err == nil {
		t.Fatal("0 != usage should fail")
	}
}
