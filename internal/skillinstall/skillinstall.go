// Package skillinstall installs Margo-provided agent skills without replacing
// locally customized files.
package skillinstall

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jjanuszczak/margo/internal/scaffold"
	"github.com/jjanuszczak/margo/internal/version"
	"gopkg.in/yaml.v3"
)

type Scope string

const (
	Project      Scope = "project"
	User         Scope = "user"
	manifestName       = ".margo-skill-manifest.yaml"
)

type Action string

const (
	Add    Action = "add"
	Update Action = "update"
	Skip   Action = "skip"
)

type Change struct {
	Path   string
	Action Action
	Reason string
}
type Plan struct {
	Target  string
	Changes []Change
}
type Manifest struct {
	Version string            `yaml:"version"`
	Margo   string            `yaml:"margo_version"`
	Files   map[string]string `yaml:"files"`
}

func Files(scope Scope) (map[string]string, error) {
	switch scope {
	case Project:
		files := map[string]string{}
		for path, content := range scaffold.AgentFiles() {
			prefix := filepath.Join(".agents", "skills", "margo-brand-theme") + string(filepath.Separator)
			if strings.HasPrefix(path, prefix) {
				files[strings.TrimPrefix(path, prefix)] = content
			}
		}
		return files, nil
	case User:
		return scaffold.BrandThemeUserSkillFiles(), nil
	default:
		return nil, fmt.Errorf("unsupported skill scope %q", scope)
	}
}

func Target(projectRoot, userHome string, scope Scope) (string, error) {
	switch scope {
	case Project:
		if projectRoot == "" {
			return "", fmt.Errorf("project scope requires a Margo project root")
		}
		return filepath.Join(projectRoot, ".agents", "skills", "margo-brand-theme"), nil
	case User:
		if userHome == "" {
			return "", fmt.Errorf("user scope requires a home directory")
		}
		return filepath.Join(userHome, ".agents", "skills", "brand-to-margo-theme"), nil
	default:
		return "", fmt.Errorf("unsupported skill scope %q", scope)
	}
}

func BuildPlan(target string, files map[string]string) (Plan, error) {
	manifest, exists, err := loadManifest(target)
	if err != nil {
		return Plan{}, err
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	plan := Plan{Target: target}
	for _, rel := range paths {
		actual, err := os.ReadFile(filepath.Join(target, rel))
		if os.IsNotExist(err) {
			plan.Changes = append(plan.Changes, Change{rel, Add, "missing Margo-provided skill file"})
			continue
		}
		if err != nil {
			return Plan{}, err
		}
		if exists && manifest.Files[filepath.ToSlash(rel)] == digest(actual) && string(actual) != files[rel] {
			plan.Changes = append(plan.Changes, Change{rel, Update, "matches recorded Margo baseline"})
			continue
		}
		if string(actual) != files[rel] {
			plan.Changes = append(plan.Changes, Change{rel, Skip, "customized file preserved"})
		}
	}
	return plan, nil
}

func Apply(plan Plan, files map[string]string) error {
	for _, change := range plan.Changes {
		if change.Action != Add && change.Action != Update {
			continue
		}
		target := filepath.Join(plan.Target, change.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(files[change.Path]), 0o644); err != nil {
			return err
		}
	}
	manifest, _, err := loadManifest(plan.Target)
	if err != nil {
		return err
	}
	if manifest.Files == nil {
		manifest = Manifest{Version: "1", Margo: version.Current(), Files: map[string]string{}}
	}
	for _, change := range plan.Changes {
		if change.Action == Add || change.Action == Update {
			manifest.Files[filepath.ToSlash(change.Path)] = digest([]byte(files[change.Path]))
		}
	}
	if len(manifest.Files) == 0 {
		return nil
	}
	manifest.Margo = version.Current()
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(plan.Target, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(plan.Target, manifestName), data, 0o644)
}

func Format(plan Plan) string {
	if len(plan.Changes) == 0 {
		return "skill install: already current\n"
	}
	var out strings.Builder
	for _, c := range plan.Changes {
		fmt.Fprintf(&out, "%s %s: %s\n", c.Action, c.Path, c.Reason)
	}
	return out.String()
}

func loadManifest(target string) (Manifest, bool, error) {
	data, err := os.ReadFile(filepath.Join(target, manifestName))
	if os.IsNotExist(err) {
		return Manifest{}, false, nil
	}
	if err != nil {
		return Manifest{}, false, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, false, err
	}
	return manifest, manifest.Version == "1", nil
}

func digest(data []byte) string { sum := sha256.Sum256(data); return fmt.Sprintf("%x", sum[:]) }
