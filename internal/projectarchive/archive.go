package projectarchive

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"margo/internal/ignore"
)

const (
	Extension       = ".margo"
	ManifestName    = "margo-archive.yaml"
	FormatVersion   = 1
	maxArchiveFiles = 10000
	maxArchiveBytes = 500 << 20
)

type Manifest struct {
	FormatVersion int    `yaml:"format_version"`
	ProjectName   string `yaml:"project_name"`
	MinMargo      string `yaml:"min_margo_version"`
}

// Pack writes a portable source archive. Generated output and local tool state
// are deliberately excluded.
func Pack(projectRoot, outputPath, margoVersion string) error {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}
	if _, err := os.Stat(filepath.Join(root, "margo.yaml")); err != nil {
		return fmt.Errorf("archive requires margo.yaml: %w", err)
	}
	if filepath.Clean(outputPath) == filepath.Join(root, ManifestName) {
		return errors.New("archive output conflicts with reserved manifest name")
	}
	matcher, err := ignore.Load(root)
	if err != nil {
		return fmt.Errorf("load ignore rules: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create archive %q: %w", outputPath, err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	defer writer.Close()

	manifest := Manifest{FormatVersion: FormatVersion, ProjectName: filepath.Base(root), MinMargo: margoVersion}
	if err := writeFile(writer, ManifestName, []byte(mustMarshalManifest(manifest)), 0o644); err != nil {
		return err
	}

	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
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
		if shouldExclude(rel, entry.IsDir()) || matcher.ShouldIgnore(rel, entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse symlink in project archive: %s", rel)
		}
		if entry.IsDir() {
			return nil
		}
		if rel == ManifestName {
			return fmt.Errorf("project contains reserved archive manifest %q", ManifestName)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeFile(writer, rel, data, 0o644)
	})
}

func mustMarshalManifest(manifest Manifest) string {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func writeFile(writer *zip.Writer, name string, data []byte, mode os.FileMode) error {
	header := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate}
	header.SetMode(mode)
	w, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func shouldExclude(rel string, isDir bool) bool {
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		switch part {
		case ".git", "dist", ".margo-backups", ".gocache":
			return true
		}
	}
	return !isDir && (strings.HasSuffix(rel, Extension) || filepath.Base(rel) == ".DS_Store")
}

// Unpack validates every member before writing any project content to dest.
func Unpack(archivePath, dest string) (Manifest, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return Manifest{}, fmt.Errorf("open archive %q: %w", archivePath, err)
	}
	defer reader.Close()
	if len(reader.File) > maxArchiveFiles {
		return Manifest{}, fmt.Errorf("archive has too many files (%d; limit %d)", len(reader.File), maxArchiveFiles)
	}

	var manifestFile *zip.File
	hasConfig := false
	var total uint64
	for _, member := range reader.File {
		if err := validateMember(member); err != nil {
			return Manifest{}, err
		}
		total += member.UncompressedSize64
		if total > maxArchiveBytes {
			return Manifest{}, fmt.Errorf("archive is too large when extracted (limit %d bytes)", maxArchiveBytes)
		}
		if member.Name == ManifestName {
			manifestFile = member
		}
		if member.Name == "margo.yaml" {
			hasConfig = true
		}
	}
	if manifestFile == nil {
		return Manifest{}, fmt.Errorf("archive is missing %s", ManifestName)
	}
	if !hasConfig {
		return Manifest{}, errors.New("archive is missing margo.yaml")
	}
	manifest, err := readManifest(manifestFile)
	if err != nil {
		return Manifest{}, err
	}
	if manifest.FormatVersion != FormatVersion {
		return Manifest{}, fmt.Errorf("unsupported archive format version %d", manifest.FormatVersion)
	}
	if err := ensureEmptyDestination(dest); err != nil {
		return Manifest{}, err
	}

	for _, member := range reader.File {
		if member.Name == ManifestName || strings.HasSuffix(member.Name, "/") {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(member.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Manifest{}, err
		}
		source, err := member.Open()
		if err != nil {
			return Manifest{}, err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			_, err = io.Copy(out, source)
			closeErr := out.Close()
			if err == nil {
				err = closeErr
			}
		}
		source.Close()
		if err != nil {
			return Manifest{}, err
		}
	}
	return manifest, nil
}

func validateMember(member *zip.File) error {
	name := strings.ReplaceAll(member.Name, "\\", "/")
	if strings.HasSuffix(name, "/") {
		name = strings.TrimSuffix(name, "/")
	}
	if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return fmt.Errorf("archive contains invalid path %q", member.Name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("archive contains unsafe path %q", member.Name)
		}
	}
	if member.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("archive contains unsupported symlink %q", member.Name)
	}
	return nil
}

func readManifest(member *zip.File) (Manifest, error) {
	reader, err := member.Open()
	if err != nil {
		return Manifest{}, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse archive manifest: %w", err)
	}
	return manifest, nil
}

func ensureEmptyDestination(dest string) error {
	info, err := os.Stat(dest)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("unpack destination is not a directory: %s", dest)
		}
		entries, err := os.ReadDir(dest)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("unpack destination is not empty: %s", dest)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(dest, 0o755)
}
