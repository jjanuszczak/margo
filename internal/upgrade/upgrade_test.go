package upgrade

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jjanuszczak/margo/internal/scaffold"
	"gopkg.in/yaml.v3"
)

func TestPlanAndApplyUpdatesUntouchedGuidance(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "deck", TargetDir: root}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("old scaffold guidance"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make the manifest baseline match the old content to emulate a prior release.
	manifestPath := filepath.Join(root, scaffold.ManifestPath)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest scaffold.Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("old scaffold guidance"))
	manifest.Files["AGENTS.md"] = fmt.Sprintf("%x", sum[:])
	raw, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasAction(plan, "AGENTS.md", Update) {
		t.Fatalf("expected AGENTS.md update: %#v", plan)
	}
	backup, err := Apply(root, plan)
	if err != nil {
		t.Fatal(err)
	}
	if backup == "" {
		t.Fatal("expected backup")
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "Margo Deck Agent Guide") {
		t.Fatalf("guidance not updated: %s", updated)
	}
	if _, err := os.Stat(filepath.Join(backup, "AGENTS.md")); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}

func TestPlanPreservesCustomizedGuidance(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "deck", TargetDir: root}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("our deck rules"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasAction(plan, "AGENTS.md", Skip) {
		t.Fatalf("expected custom file skip: %#v", plan)
	}
	if _, err := Apply(root, plan); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(path)
	if string(actual) != "our deck rules" {
		t.Fatalf("custom content overwritten: %q", actual)
	}
}

func TestPlanAddsMissingErrorTriageSkillToOlderDeck(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "deck", TargetDir: root}); err != nil {
		t.Fatal(err)
	}

	for path := range scaffold.AgentFiles() {
		if strings.Contains(path, "margo-error-triage") {
			if err := os.Remove(filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
		}
	}

	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	for path := range scaffold.AgentFiles() {
		if strings.Contains(path, "margo-error-triage") && !hasAction(plan, path, Add) {
			t.Errorf("expected missing error-triage resource %q to be added: %#v", path, plan)
		}
	}

	if _, err := Apply(root, plan); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".agents", "skills", "margo-error-triage", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "name: margo-error-triage") {
		t.Fatalf("unexpected error-triage skill: %s", raw)
	}
}

func hasAction(plan Plan, path string, action Action) bool {
	for _, c := range plan.Changes {
		if c.Path == path && c.Action == action {
			return true
		}
	}
	return false
}
