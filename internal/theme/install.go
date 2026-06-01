package theme

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type InstallOptions struct {
	ProjectRoot string
	Repo        string
	Ref         string
	Name        string
}

type InstalledTheme struct {
	Name   string
	Source *Source
}

func Install(opts InstallOptions) (InstalledTheme, error) {
	if strings.TrimSpace(opts.ProjectRoot) == "" {
		return InstalledTheme{}, errors.New("project root is required")
	}
	if strings.TrimSpace(opts.Repo) == "" {
		return InstalledTheme{}, errors.New("theme repo is required")
	}

	cloneDir, err := os.MkdirTemp("", "margo-theme-clone-*")
	if err != nil {
		return InstalledTheme{}, fmt.Errorf("create temporary clone directory: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	if err := runGit("", "clone", opts.Repo, cloneDir); err != nil {
		return InstalledTheme{}, fmt.Errorf("clone theme repo %q: %w", opts.Repo, err)
	}
	if strings.TrimSpace(opts.Ref) != "" {
		if err := runGit(cloneDir, "checkout", opts.Ref); err != nil {
			return InstalledTheme{}, fmt.Errorf("checkout theme ref %q: %w", opts.Ref, err)
		}
	}

	resolvedRef, err := gitOutput(cloneDir, "rev-parse", "HEAD")
	if err != nil {
		return InstalledTheme{}, fmt.Errorf("resolve installed theme revision: %w", err)
	}

	meta, err := loadFromRootDir(cloneDir, filepath.Base(cloneDir))
	if err != nil {
		return InstalledTheme{}, fmt.Errorf("load cloned theme: %w", err)
	}

	themeName := strings.TrimSpace(opts.Name)
	if themeName == "" {
		themeName = strings.TrimSpace(meta.Name)
	}
	if themeName == "" {
		return InstalledTheme{}, errors.New("installed theme name is empty")
	}

	targetDir := filepath.Join(opts.ProjectRoot, ThemesDirName, themeName)
	if _, err := os.Stat(targetDir); err == nil {
		return InstalledTheme{}, fmt.Errorf("theme already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return InstalledTheme{}, fmt.Errorf("stat target theme directory %q: %w", targetDir, err)
	}

	if err := copyThemeTree(cloneDir, targetDir); err != nil {
		return InstalledTheme{}, fmt.Errorf("copy theme into deck: %w", err)
	}

	source := &Source{
		Type:        "git",
		Repo:        opts.Repo,
		Ref:         strings.TrimSpace(opts.Ref),
		ResolvedRef: strings.TrimSpace(resolvedRef),
	}
	if err := writeSourceMetadata(filepath.Join(targetDir, ThemeMetadataFile), source, themeName); err != nil {
		_ = os.RemoveAll(targetDir)
		return InstalledTheme{}, fmt.Errorf("write installed theme metadata: %w", err)
	}

	if _, err := loadFromRootDir(targetDir, themeName); err != nil {
		_ = os.RemoveAll(targetDir)
		return InstalledTheme{}, fmt.Errorf("validate installed theme: %w", err)
	}

	return InstalledTheme{Name: themeName, Source: source}, nil
}

func List(projectRoot string) ([]InstalledTheme, error) {
	root := filepath.Join(projectRoot, ThemesDirName)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read themes directory %q: %w", root, err)
	}

	result := make([]InstalledTheme, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := Load(projectRoot, entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, InstalledTheme{
			Name:   entry.Name(),
			Source: meta.Source,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func copyThemeTree(srcRoot, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return os.MkdirAll(dstRoot, 0o755)
		}
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) > 0 && parts[0] == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(dstRoot, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyFile(path, targetPath)
	})
}

func copyFile(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, data, 0o644)
}

func writeSourceMetadata(metadataPath string, source *Source, fallbackName string) error {
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("read theme metadata %q: %w", metadataPath, err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse theme metadata %q: %w", metadataPath, err)
	}
	if strings.TrimSpace(fallbackName) != "" {
		if _, ok := doc["name"]; !ok {
			doc["name"] = fallbackName
		}
	}
	doc["source"] = source

	updated, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal theme metadata %q: %w", metadataPath, err)
	}
	return os.WriteFile(metadataPath, updated, 0o644)
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return "", fmt.Errorf("%w: %s", err, message)
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
