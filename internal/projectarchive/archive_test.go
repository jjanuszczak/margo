package projectarchive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPackAndUnpackPortableProject(t *testing.T) {
	parent := t.TempDir()
	projectRoot := filepath.Join(parent, "deck")
	writeArchiveFile(t, filepath.Join(projectRoot, "margo.yaml"), "version: 1\n")
	writeArchiveFile(t, filepath.Join(projectRoot, "slides", "01-title", "index.md"), "# Title\n")
	writeArchiveFile(t, filepath.Join(projectRoot, "themes", "brand", "theme.yaml"), "name: brand\n")
	writeArchiveFile(t, filepath.Join(projectRoot, "archetypes", "default", "archetype.yaml"), "name: default\n")
	writeArchiveFile(t, filepath.Join(projectRoot, "dist", "html", "index.html"), "generated")
	writeArchiveFile(t, filepath.Join(projectRoot, ".git", "config"), "gitdir")
	writeArchiveFile(t, filepath.Join(projectRoot, ".margo-backups", "old"), "backup")
	writeArchiveFile(t, filepath.Join(projectRoot, "assets", ".DS_Store"), "noise")

	archivePath := filepath.Join(parent, "deck.margo")
	if err := Pack(projectRoot, archivePath, "0.1.0"); err != nil {
		t.Fatalf("Pack returned error: %v", err)
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	var names []string
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	reader.Close()
	sort.Strings(names)
	for _, want := range []string{ManifestName, "margo.yaml", "slides/01-title/index.md", "themes/brand/theme.yaml"} {
		if !contains(names, want) {
			t.Fatalf("expected archive to contain %q, got %#v", want, names)
		}
	}
	for _, forbidden := range []string{"dist/html/index.html", ".git/config", ".margo-backups/old", "assets/.DS_Store"} {
		if contains(names, forbidden) {
			t.Fatalf("archive must exclude %q, got %#v", forbidden, names)
		}
	}

	destination := filepath.Join(parent, "restored")
	manifest, err := Unpack(archivePath, destination)
	if err != nil {
		t.Fatalf("Unpack returned error: %v", err)
	}
	if manifest.FormatVersion != FormatVersion || manifest.ProjectName != "deck" || manifest.MinMargo != "0.1.0" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	for _, rel := range []string{"margo.yaml", "slides/01-title/index.md", "themes/brand/theme.yaml"} {
		if _, err := os.Stat(filepath.Join(destination, rel)); err != nil {
			t.Fatalf("expected restored %s: %v", rel, err)
		}
	}
}

func TestUnpackRejectsUnsafeMembers(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "unsafe.margo")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	if err := writeFile(writer, ManifestName, []byte("format_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(writer, "../escape.md", []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	file.Close()

	_, err = Unpack(archivePath, filepath.Join(t.TempDir(), "restored"))
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
}

func TestUnpackRejectsNonEmptyDestination(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "empty.margo")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	if err := writeFile(writer, ManifestName, []byte("format_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(writer, "margo.yaml", []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	file.Close()
	dest := t.TempDir()
	writeArchiveFile(t, filepath.Join(dest, "existing"), "x")
	if _, err := Unpack(archivePath, dest); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected non-empty destination error, got %v", err)
	}
}

func writeArchiveFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
