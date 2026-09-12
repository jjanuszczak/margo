package skillinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectSkillInstallPreservesCustomFiles(t *testing.T) {
	root := t.TempDir()
	target, err := Target(root, "", Project)
	if err != nil {
		t.Fatal(err)
	}
	files, err := Files(Project)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(target, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != len(files) {
		t.Fatalf("expected missing files, got %#v", plan)
	}
	if err := Apply(plan, files); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	custom := filepath.Join(target, "SKILL.md")
	if err := os.WriteFile(custom, []byte("custom workflow"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = BuildPlan(target, files)
	if err != nil {
		t.Fatal(err)
	}
	if !has(plan, "SKILL.md", Skip) {
		t.Fatalf("expected preserved customization: %#v", plan)
	}
	if err := Apply(plan, files); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(custom)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "custom workflow" {
		t.Fatalf("custom file overwritten: %q", actual)
	}
}

func TestUserSkillUsesDistinctNameAndTracksManifest(t *testing.T) {
	home := t.TempDir()
	target, err := Target("", home, User)
	if err != nil {
		t.Fatal(err)
	}
	files, err := Files(User)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["SKILL.md"], "name: brand-to-margo-theme") {
		t.Fatal("expected distinct global skill name")
	}
	if !strings.Contains(files["references/component-contract.md"], "Brand component contract") {
		t.Fatal("expected global skill to include component contract")
	}
	plan, err := BuildPlan(target, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(plan, files); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, manifestName)); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	plan, err = BuildPlan(target, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 0 {
		t.Fatalf("expected no-op plan, got %#v", plan)
	}
}

func has(plan Plan, path string, action Action) bool {
	for _, change := range plan.Changes {
		if change.Path == path && change.Action == action {
			return true
		}
	}
	return false
}
