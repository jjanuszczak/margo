// Package upgrade safely refreshes Margo-managed project guidance. It never
// rewrites slides, configuration, assets, or themes.
package upgrade

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jjanuszczak/margo/internal/scaffold"
	"github.com/jjanuszczak/margo/internal/version"
	"gopkg.in/yaml.v3"
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

type Plan struct{ Changes []Change }

func (p Plan) HasWrites() bool {
	for _, change := range p.Changes {
		if change.Action == Add || change.Action == Update {
			return true
		}
	}
	return false
}

func BuildPlan(root string) (Plan, error) {
	manifest, hasManifest, err := loadManifest(root)
	if err != nil {
		return Plan{}, err
	}
	files := scaffold.AgentFiles()
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	plan := Plan{}
	for _, rel := range paths {
		path := filepath.Join(root, rel)
		actual, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			plan.Changes = append(plan.Changes, Change{Path: filepath.ToSlash(rel), Action: Add, Reason: "missing Margo-managed guidance"})
			continue
		}
		if err != nil {
			return Plan{}, fmt.Errorf("read %s: %w", rel, err)
		}
		baseline, tracked := manifest.Files[filepath.ToSlash(rel)]
		if hasManifest && tracked && digest(actual) == baseline {
			if string(actual) != files[rel] {
				plan.Changes = append(plan.Changes, Change{Path: filepath.ToSlash(rel), Action: Update, Reason: "matches recorded scaffold baseline"})
			}
			continue
		}
		if string(actual) != files[rel] {
			reason := "customized or legacy file preserved"
			if !hasManifest {
				reason = "legacy project file preserved"
			}
			plan.Changes = append(plan.Changes, Change{Path: filepath.ToSlash(rel), Action: Skip, Reason: reason})
		}
	}
	if !hasManifest {
		plan.Changes = append(plan.Changes, Change{Path: scaffold.ManifestPath, Action: Add, Reason: "record upgrade-managed scaffold baselines"})
	}
	return plan, nil
}

func Apply(root string, plan Plan) (string, error) {
	if !plan.HasWrites() && !containsManifest(plan) {
		return "", nil
	}
	backup := filepath.Join(root, ".margo-backups", "upgrade-"+time.Now().UTC().Format("20060102T150405Z"))
	for _, change := range plan.Changes {
		if change.Action != Update {
			continue
		}
		source := filepath.Join(root, filepath.FromSlash(change.Path))
		target := filepath.Join(backup, filepath.FromSlash(change.Path))
		data, err := os.ReadFile(source)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return "", err
		}
	}
	files := scaffold.AgentFiles()
	for _, change := range plan.Changes {
		if change.Action != Add && change.Action != Update {
			continue
		}
		if change.Path == scaffold.ManifestPath {
			continue
		}
		data := files[filepath.FromSlash(change.Path)]
		target := filepath.Join(root, filepath.FromSlash(change.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, []byte(data), 0o644); err != nil {
			return "", err
		}
	}
	if err := refreshManifest(root, plan); err != nil {
		return "", err
	}
	return backup, nil
}

func containsManifest(plan Plan) bool {
	for _, c := range plan.Changes {
		if c.Path == scaffold.ManifestPath {
			return true
		}
	}
	return false
}

func loadManifest(root string) (scaffold.Manifest, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, scaffold.ManifestPath))
	if os.IsNotExist(err) {
		return scaffold.Manifest{}, false, nil
	}
	if err != nil {
		return scaffold.Manifest{}, false, err
	}
	var manifest scaffold.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return scaffold.Manifest{}, false, fmt.Errorf("parse scaffold manifest: %w", err)
	}
	if manifest.Version != "1" {
		return scaffold.Manifest{}, false, fmt.Errorf("unsupported scaffold manifest version %q", manifest.Version)
	}
	return manifest, true, nil
}

func refreshManifest(root string, plan Plan) error {
	manifest, exists, err := loadManifest(root)
	if err != nil {
		return err
	}
	if !exists {
		manifest = scaffold.Manifest{Version: "1", Files: map[string]string{}}
	}
	if manifest.Files == nil {
		manifest.Files = map[string]string{}
	}
	for _, change := range plan.Changes {
		if change.Action != Add && change.Action != Update || change.Path == scaffold.ManifestPath {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(change.Path)))
		if err != nil {
			return err
		}
		manifest.Files[change.Path] = digest(data)
	}
	manifest.Margo = version.Current()
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	target := filepath.Join(root, scaffold.ManifestPath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func digest(data []byte) string { sum := sha256.Sum256(data); return fmt.Sprintf("%x", sum[:]) }

func Format(plan Plan) string {
	if len(plan.Changes) == 0 {
		return "upgrade: project scaffolding is current\n"
	}
	var out strings.Builder
	for _, c := range plan.Changes {
		fmt.Fprintf(&out, "%s %s: %s\n", c.Action, c.Path, c.Reason)
	}
	return out.String()
}
