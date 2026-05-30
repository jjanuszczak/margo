package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndMatch(t *testing.T) {
	projectRoot := t.TempDir()
	content := "# comment\nignored-slide/\n*.tmp\nassets/generated.svg\n"
	if err := os.WriteFile(filepath.Join(projectRoot, DefaultFilename), []byte(content), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}

	matcher, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	cases := []struct {
		rel    string
		isDir  bool
		ignore bool
	}{
		{"slides/ignored-slide", true, true},
		{"slides/ignored-slide/index.md", false, true},
		{"slides/other/tmp.tmp", false, true},
		{"assets/generated.svg", false, true},
		{"assets/keep.svg", false, false},
		{"slides/visible/index.md", false, false},
	}

	for _, tc := range cases {
		if got := matcher.ShouldIgnore(tc.rel, tc.isDir); got != tc.ignore {
			t.Fatalf("ShouldIgnore(%q, %t) = %t, want %t", tc.rel, tc.isDir, got, tc.ignore)
		}
	}
}
