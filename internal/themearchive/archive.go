// Package themearchive packages one vendored theme for offline handoff.
package themearchive

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jjanuszczak/margo/internal/theme"
	"gopkg.in/yaml.v3"
)

const (
	Extension       = ".margot"
	ManifestName    = "margot-theme-archive.yaml"
	FormatVersion   = 1
	maxArchiveFiles = 10000
	maxArchiveBytes = 500 << 20
)

type Manifest struct {
	FormatVersion int       `yaml:"format_version"`
	ThemeName     string    `yaml:"theme_name"`
	ThemeVersion  string    `yaml:"theme_version"`
	MinMargo      string    `yaml:"min_margo_version"`
	CreatedAt     time.Time `yaml:"created_at"`
	PayloadSHA256 string    `yaml:"payload_sha256"`
}

type ImportedTheme struct {
	Name     string
	Manifest Manifest
}

type archiveFile struct {
	name string
	data []byte
}

// Pack validates and packages one theme from a deck project. It never includes
// deck content or other installed themes.
func Pack(projectRoot, themeName, outputPath, margoVersion string) (Manifest, error) {
	themeName = strings.TrimSpace(themeName)
	if themeName == "" {
		return Manifest{}, errors.New("theme name is required")
	}
	if !strings.EqualFold(filepath.Ext(outputPath), Extension) {
		return Manifest{}, fmt.Errorf("theme archive output must use %s", Extension)
	}
	meta, err := theme.Load(projectRoot, themeName)
	if err != nil {
		return Manifest{}, fmt.Errorf("load theme %q: %w", themeName, err)
	}
	files, err := collectThemeFiles(meta.RootDir)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		FormatVersion: FormatVersion,
		ThemeName:     meta.Name,
		ThemeVersion:  meta.Version,
		MinMargo:      margoVersion,
		CreatedAt:     time.Now().UTC(),
		PayloadSHA256: payloadChecksum(files),
	}
	if err := writeArchive(outputPath, manifest, files); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Import validates an archive, stages it beneath the destination project's
// themes directory, then atomically exposes it at the requested local name.
func Import(projectRoot, archivePath, localName, margoVersion string) (ImportedTheme, error) {
	if !strings.EqualFold(filepath.Ext(archivePath), Extension) {
		return ImportedTheme{}, fmt.Errorf("theme archive must use %s", Extension)
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return ImportedTheme{}, fmt.Errorf("open theme archive %q: %w", archivePath, err)
	}
	defer reader.Close()

	manifest, files, err := validateArchive(reader.File)
	if err != nil {
		return ImportedTheme{}, err
	}
	if manifest.FormatVersion != FormatVersion {
		return ImportedTheme{}, fmt.Errorf("unsupported theme archive format version %d", manifest.FormatVersion)
	}
	if strings.TrimSpace(manifest.MinMargo) == "" {
		return ImportedTheme{}, errors.New("theme archive manifest is missing min_margo_version")
	}
	if !compatibleVersion(margoVersion, manifest.MinMargo) {
		return ImportedTheme{}, fmt.Errorf("theme archive requires Margo %s or later (current version %s)", manifest.MinMargo, margoVersion)
	}
	if payloadChecksum(files) != manifest.PayloadSHA256 {
		return ImportedTheme{}, errors.New("theme archive payload checksum does not match manifest")
	}

	name := strings.TrimSpace(localName)
	if name == "" {
		name = strings.TrimSpace(manifest.ThemeName)
	}
	if name == "" {
		return ImportedTheme{}, errors.New("theme archive manifest is missing theme_name")
	}
	if filepath.Base(name) != name || name == "." {
		return ImportedTheme{}, fmt.Errorf("invalid local theme name %q", name)
	}

	themesDir := filepath.Join(projectRoot, theme.ThemesDirName)
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		return ImportedTheme{}, fmt.Errorf("create themes directory: %w", err)
	}
	targetDir := filepath.Join(themesDir, name)
	if _, err := os.Stat(targetDir); err == nil {
		return ImportedTheme{}, fmt.Errorf("theme already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return ImportedTheme{}, fmt.Errorf("stat target theme directory %q: %w", targetDir, err)
	}

	stagedDir, err := os.MkdirTemp(themesDir, name+".staged-*")
	if err != nil {
		return ImportedTheme{}, fmt.Errorf("create theme staging directory: %w", err)
	}
	defer os.RemoveAll(stagedDir)
	if err := extractFiles(stagedDir, files); err != nil {
		return ImportedTheme{}, err
	}
	stagedMeta, err := theme.ValidateRoot(stagedDir, manifest.ThemeName)
	if err != nil {
		return ImportedTheme{}, fmt.Errorf("validate staged theme: %w", err)
	}
	if stagedMeta.Name != manifest.ThemeName || stagedMeta.Version != manifest.ThemeVersion {
		return ImportedTheme{}, errors.New("theme archive manifest does not match theme.yaml")
	}
	if err := writeArchiveSource(filepath.Join(stagedDir, theme.ThemeMetadataFile), manifest, name); err != nil {
		return ImportedTheme{}, err
	}
	if _, err := theme.ValidateRoot(stagedDir, name); err != nil {
		return ImportedTheme{}, fmt.Errorf("validate staged theme: %w", err)
	}
	if err := validateStagedTheme(stagedDir); err != nil {
		return ImportedTheme{}, err
	}
	if err := os.Rename(stagedDir, targetDir); err != nil {
		return ImportedTheme{}, fmt.Errorf("install theme archive: %w", err)
	}
	return ImportedTheme{Name: name, Manifest: manifest}, nil
}

func collectThemeFiles(root string) ([]archiveFile, error) {
	var files []archiveFile
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if shouldExclude(rel, entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse symlink in theme archive: %s", rel)
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, archiveFile{name: rel, data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, nil
}

func shouldExclude(rel string, isDir bool) bool {
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		switch part {
		case ".git", "dist", ".margo-backups", ".gocache":
			return true
		}
	}
	if isDir {
		return false
	}
	base := filepath.Base(rel)
	return base == ".DS_Store" || strings.HasSuffix(rel, ".margo") || strings.HasSuffix(rel, Extension)
}

func writeArchive(path string, manifest Manifest, files []archiveFile) error {
	output, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create theme archive %q: %w", path, err)
	}
	defer output.Close()
	writer := zip.NewWriter(output)
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	if err := writeFile(writer, ManifestName, manifestBytes); err != nil {
		return err
	}
	for _, file := range files {
		if err := writeFile(writer, file.name, file.data); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize theme archive %q: %w", path, err)
	}
	return nil
}

func writeFile(writer *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o644)
	w, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func validateArchive(members []*zip.File) (Manifest, []archiveFile, error) {
	if len(members) > maxArchiveFiles {
		return Manifest{}, nil, fmt.Errorf("theme archive has too many files (%d; limit %d)", len(members), maxArchiveFiles)
	}
	seen := map[string]struct{}{}
	var manifestFile *zip.File
	var files []archiveFile
	var total uint64
	for _, member := range members {
		if err := validateMember(member, seen); err != nil {
			return Manifest{}, nil, err
		}
		if strings.HasSuffix(member.Name, "/") {
			continue
		}
		total += member.UncompressedSize64
		if total > maxArchiveBytes {
			return Manifest{}, nil, fmt.Errorf("theme archive is too large when extracted (limit %d bytes)", maxArchiveBytes)
		}
		if member.Name == ManifestName {
			manifestFile = member
			continue
		}
		data, err := readMember(member)
		if err != nil {
			return Manifest{}, nil, err
		}
		files = append(files, archiveFile{name: member.Name, data: data})
	}
	if manifestFile == nil {
		return Manifest{}, nil, fmt.Errorf("theme archive is missing %s", ManifestName)
	}
	if !containsFile(files, theme.ThemeMetadataFile) {
		return Manifest{}, nil, errors.New("theme archive is missing theme.yaml")
	}
	manifestRaw, err := readMember(manifestFile)
	if err != nil {
		return Manifest{}, nil, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(manifestRaw, &manifest); err != nil {
		return Manifest{}, nil, fmt.Errorf("parse theme archive manifest: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return manifest, files, nil
}

func compatibleVersion(current, minimum string) bool {
	currentParts, currentOK := parseVersion(current)
	minimumParts, minimumOK := parseVersion(minimum)
	if !currentOK || !minimumOK {
		return true
	}
	for i := range currentParts {
		if currentParts[i] != minimumParts[i] {
			return currentParts[i] > minimumParts[i]
		}
	}
	return true
}

func parseVersion(value string) ([3]int, bool) {
	var parts [3]int
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if strings.Contains(value, "-") {
		return parts, false
	}
	segments := strings.Split(value, ".")
	if len(segments) != 3 {
		return parts, false
	}
	for i, segment := range segments {
		for _, runeValue := range segment {
			if runeValue < '0' || runeValue > '9' {
				return parts, false
			}
		}
		for _, runeValue := range segment {
			parts[i] = parts[i]*10 + int(runeValue-'0')
		}
	}
	return parts, true
}

func validateMember(member *zip.File, seen map[string]struct{}) error {
	name := strings.ReplaceAll(member.Name, "\\", "/")
	if strings.HasSuffix(name, "/") {
		name = strings.TrimSuffix(name, "/")
	}
	if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return fmt.Errorf("theme archive contains invalid path %q", member.Name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("theme archive contains unsafe path %q", member.Name)
		}
	}
	if member.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("theme archive contains unsupported symlink %q", member.Name)
	}
	if _, ok := seen[name]; ok {
		return fmt.Errorf("theme archive contains duplicate path %q", member.Name)
	}
	seen[name] = struct{}{}
	return nil
}

func readMember(member *zip.File) ([]byte, error) {
	reader, err := member.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, int64(maxArchiveBytes)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) != member.UncompressedSize64 {
		return nil, fmt.Errorf("read theme archive member %q", member.Name)
	}
	return data, nil
}

func payloadChecksum(files []archiveFile) string {
	hash := sha256.New()
	for _, file := range files {
		hash.Write([]byte(file.name))
		hash.Write([]byte{0})
		hash.Write(file.data)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func containsFile(files []archiveFile, name string) bool {
	for _, file := range files {
		if file.name == name {
			return true
		}
	}
	return false
}

func extractFiles(destination string, files []archiveFile) error {
	for _, file := range files {
		target := filepath.Join(destination, filepath.FromSlash(file.name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, file.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func validateStagedTheme(stagedDir string) error {
	metadataPath := filepath.Join(stagedDir, theme.ThemeMetadataFile)
	if _, err := os.Stat(metadataPath); err != nil {
		return fmt.Errorf("validate staged theme: %w", err)
	}
	return nil
}

func writeArchiveSource(metadataPath string, manifest Manifest, localName string) error {
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	doc["name"] = localName
	doc["source"] = theme.Source{
		Type:                 "archive",
		ArchiveFormat:        "margot",
		ArchiveSHA256:        manifest.PayloadSHA256,
		ImportedThemeName:    manifest.ThemeName,
		ImportedThemeVersion: manifest.ThemeVersion,
	}
	updated, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath, updated, 0o644)
}
