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

	cloneDir, resolvedRef, err := cloneThemeRepo(opts.Repo, opts.Ref)
	if err != nil {
		return InstalledTheme{}, err
	}
	defer os.RemoveAll(cloneDir)

	return installFromClonedTheme(cloneDir, resolvedRef, opts)
}

func Update(projectRoot, themeName string) (InstalledTheme, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return InstalledTheme{}, errors.New("project root is required")
	}
	themeName = strings.TrimSpace(themeName)
	if themeName == "" {
		return InstalledTheme{}, errors.New("theme name is required")
	}

	meta, err := Load(projectRoot, themeName)
	if err != nil {
		return InstalledTheme{}, fmt.Errorf("load installed theme: %w", err)
	}
	if meta.Source == nil {
		return InstalledTheme{}, fmt.Errorf("theme %q has no recorded source metadata", themeName)
	}
	if meta.Source.Type != "git" {
		return InstalledTheme{}, fmt.Errorf("theme %q source type %q is not updatable", themeName, meta.Source.Type)
	}
	if strings.TrimSpace(meta.Source.Repo) == "" {
		return InstalledTheme{}, fmt.Errorf("theme %q source repo is empty", themeName)
	}

	cloneDir, resolvedRef, err := cloneThemeRepo(meta.Source.Repo, meta.Source.Ref)
	if err != nil {
		return InstalledTheme{}, err
	}
	defer os.RemoveAll(cloneDir)

	stagedDir, source, err := stageInstalledTheme(cloneDir, resolvedRef, InstallOptions{
		ProjectRoot: projectRoot,
		Repo:        meta.Source.Repo,
		Ref:         meta.Source.Ref,
		Name:        themeName,
	})
	if err != nil {
		return InstalledTheme{}, err
	}
	defer os.RemoveAll(stagedDir)

	targetDir := filepath.Join(projectRoot, ThemesDirName, themeName)
	backupDir := targetDir + ".bak"
	_ = os.RemoveAll(backupDir)
	if err := os.Rename(targetDir, backupDir); err != nil {
		return InstalledTheme{}, fmt.Errorf("backup installed theme %q: %w", themeName, err)
	}
	restoreBackup := true
	defer func() {
		if restoreBackup {
			_ = os.RemoveAll(targetDir)
			_ = os.Rename(backupDir, targetDir)
		}
	}()
	if err := os.Rename(stagedDir, targetDir); err != nil {
		return InstalledTheme{}, fmt.Errorf("replace installed theme %q: %w", themeName, err)
	}
	restoreBackup = false
	_ = os.RemoveAll(backupDir)

	return InstalledTheme{Name: themeName, Source: source}, nil
}

func cloneThemeRepo(repo, ref string) (string, string, error) {
	cloneDir, err := os.MkdirTemp("", "margo-theme-clone-*")
	if err != nil {
		return "", "", fmt.Errorf("create temporary clone directory: %w", err)
	}

	if err := runGit("", "clone", repo, cloneDir); err != nil {
		_ = os.RemoveAll(cloneDir)
		return "", "", fmt.Errorf("clone theme repo %q: %w", repo, err)
	}
	if strings.TrimSpace(ref) != "" {
		if err := runGit(cloneDir, "checkout", ref); err != nil {
			_ = os.RemoveAll(cloneDir)
			return "", "", fmt.Errorf("checkout theme ref %q: %w", ref, err)
		}
	}

	resolvedRef, err := gitOutput(cloneDir, "rev-parse", "HEAD")
	if err != nil {
		_ = os.RemoveAll(cloneDir)
		return "", "", fmt.Errorf("resolve installed theme revision: %w", err)
	}
	return cloneDir, resolvedRef, nil
}

func installFromClonedTheme(cloneDir, resolvedRef string, opts InstallOptions) (InstalledTheme, error) {
	themeName, err := installedThemeName(cloneDir, opts.Name)
	if err != nil {
		return InstalledTheme{}, err
	}
	parentDir := filepath.Join(opts.ProjectRoot, ThemesDirName)
	targetDir := filepath.Join(parentDir, themeName)
	if _, err := os.Stat(targetDir); err == nil {
		return InstalledTheme{}, fmt.Errorf("theme already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return InstalledTheme{}, fmt.Errorf("stat target theme directory %q: %w", targetDir, err)
	}

	stagedDir, source, err := stageInstalledTheme(cloneDir, resolvedRef, InstallOptions{
		ProjectRoot: opts.ProjectRoot,
		Repo:        opts.Repo,
		Ref:         opts.Ref,
		Name:        themeName,
	})
	if err != nil {
		return InstalledTheme{}, err
	}
	defer os.RemoveAll(stagedDir)

	if err := os.Rename(stagedDir, targetDir); err != nil {
		return InstalledTheme{}, fmt.Errorf("move installed theme into deck: %w", err)
	}

	return InstalledTheme{Name: themeName, Source: source}, nil
}

func stageInstalledTheme(cloneDir, resolvedRef string, opts InstallOptions) (string, *Source, error) {
	themeName, err := installedThemeName(cloneDir, opts.Name)
	if err != nil {
		return "", nil, err
	}

	parentDir := filepath.Join(opts.ProjectRoot, ThemesDirName)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create themes directory %q: %w", parentDir, err)
	}
	targetDir, err := os.MkdirTemp(parentDir, themeName+".staged-*")
	if err != nil {
		return "", nil, fmt.Errorf("create staged theme directory: %w", err)
	}

	if err := copyThemeTree(cloneDir, targetDir); err != nil {
		_ = os.RemoveAll(targetDir)
		return "", nil, fmt.Errorf("copy theme into deck: %w", err)
	}

	source := &Source{
		Type:        "git",
		Repo:        strings.TrimSpace(opts.Repo),
		Ref:         strings.TrimSpace(opts.Ref),
		ResolvedRef: strings.TrimSpace(resolvedRef),
	}
	if err := writeSourceMetadata(filepath.Join(targetDir, ThemeMetadataFile), source, themeName); err != nil {
		_ = os.RemoveAll(targetDir)
		return "", nil, fmt.Errorf("write installed theme metadata: %w", err)
	}

	if _, err := loadFromRootDir(targetDir, themeName); err != nil {
		_ = os.RemoveAll(targetDir)
		return "", nil, fmt.Errorf("validate installed theme: %w", err)
	}

	return targetDir, source, nil
}

func installedThemeName(cloneDir, requestedName string) (string, error) {
	meta, err := loadFromRootDir(cloneDir, filepath.Base(cloneDir))
	if err != nil {
		return "", fmt.Errorf("load cloned theme: %w", err)
	}

	themeName := strings.TrimSpace(requestedName)
	if themeName == "" {
		themeName = strings.TrimSpace(meta.Name)
	}
	if themeName == "" {
		return "", errors.New("installed theme name is empty")
	}
	return themeName, nil
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
