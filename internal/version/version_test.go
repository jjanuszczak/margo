package version

import "testing"

func TestCurrentFallsBackToDevelopmentVersion(t *testing.T) {
	original := Version
	Version = "0.0.0-dev"
	t.Cleanup(func() { Version = original })

	if got := Current(); got != "0.0.0-dev" {
		t.Fatalf("Current() = %q, want %q", got, "0.0.0-dev")
	}
}

func TestCurrentPrefersLinkerVersion(t *testing.T) {
	original := Version
	Version = "v1.2.3"
	t.Cleanup(func() { Version = original })

	if got := Current(); got != "v1.2.3" {
		t.Fatalf("Current() = %q, want %q", got, "v1.2.3")
	}
}
