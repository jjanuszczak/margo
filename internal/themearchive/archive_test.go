package themearchive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jjanuszczak/margo/internal/theme"
)

func TestPackAndImportThemeArchive(t *testing.T) {
	parent := t.TempDir()
	sourceProject := filepath.Join(parent, "source")
	writeThemeFile(t, filepath.Join(sourceProject, "margo.yaml"), "deck:\n  title: Source\ntheme:\n  name: brand\n")
	writeTheme(t, sourceProject, "brand")
	writeThemeFile(t, filepath.Join(sourceProject, "themes", "brand", "assets", "logo.svg"), "<svg/>")
	writeTheme(t, sourceProject, "other")
	writeThemeFile(t, filepath.Join(sourceProject, "slides", "01-title", "index.md"), "# Do not package")
	writeThemeFile(t, filepath.Join(sourceProject, "themes", "brand", ".git", "config"), "gitdir")
	writeThemeFile(t, filepath.Join(sourceProject, "themes", "brand", "dist", "generated.css"), "generated")

	archivePath := filepath.Join(parent, "brand.margot")
	manifest, err := Pack(sourceProject, "brand", archivePath, "0.2.0")
	if err != nil {
		t.Fatalf("Pack returned error: %v", err)
	}
	if manifest.ThemeName != "brand" || manifest.ThemeVersion != "1.2.0" || manifest.MinMargo != "0.2.0" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, member := range reader.File {
		names = append(names, member.Name)
	}
	reader.Close()
	sort.Strings(names)
	for _, expected := range []string{ManifestName, "theme.yaml", "layouts/default.html", "assets/logo.svg"} {
		if !contains(names, expected) {
			t.Fatalf("archive is missing %q: %#v", expected, names)
		}
	}
	for _, unexpected := range []string{"slides/01-title/index.md", ".git/config", "dist/generated.css"} {
		if contains(names, unexpected) {
			t.Fatalf("archive includes %q: %#v", unexpected, names)
		}
	}

	targetProject := filepath.Join(parent, "target")
	installed, err := Import(targetProject, archivePath, "client-brand", "0.2.0")
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if installed.Name != "client-brand" {
		t.Fatalf("expected local name client-brand, got %q", installed.Name)
	}
	if _, err := os.Stat(filepath.Join(targetProject, "themes", "client-brand", "assets", "logo.svg")); err != nil {
		t.Fatalf("expected imported asset: %v", err)
	}
	meta, err := theme.Load(targetProject, "client-brand")
	if err != nil {
		t.Fatalf("load imported theme: %v", err)
	}
	if meta.Source == nil || meta.Source.Type != "archive" || meta.Source.ArchiveSHA256 != manifest.PayloadSHA256 {
		t.Fatalf("expected archive provenance, got %#v", meta.Source)
	}
	if meta.Name != "client-brand" {
		t.Fatalf("expected local theme metadata name, got %q", meta.Name)
	}
}

func TestImportRejectsUnsafePathBeforeWriting(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "unsafe.margot")
	manifest := Manifest{FormatVersion: FormatVersion, ThemeName: "brand", ThemeVersion: "1.0.0", MinMargo: "0.1.0", PayloadSHA256: "ignored"}
	writeCustomArchive(t, archivePath, manifest, map[string]string{
		"theme.yaml":           "name: brand\n",
		"../outside.html":      "bad",
		"layouts/default.html": "layout",
	})
	target := t.TempDir()
	_, err := Import(target, archivePath, "", "0.1.0")
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "themes", "brand")); !os.IsNotExist(err) {
		t.Fatalf("import must not write target theme, stat error: %v", err)
	}
}

func TestImportRejectsChecksumMismatch(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "checksum.margot")
	manifest := Manifest{FormatVersion: FormatVersion, ThemeName: "brand", ThemeVersion: "1.0.0", MinMargo: "0.1.0", PayloadSHA256: "wrong"}
	writeCustomArchive(t, archivePath, manifest, map[string]string{
		"theme.yaml":           "name: brand\n",
		"layouts/default.html": "layout",
	})
	_, err := Import(t.TempDir(), archivePath, "", "0.1.0")
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum error, got %v", err)
	}
}

func TestImportDoesNotOverwriteExistingTheme(t *testing.T) {
	parent := t.TempDir()
	sourceProject := filepath.Join(parent, "source")
	writeTheme(t, sourceProject, "brand")
	archivePath := filepath.Join(parent, "brand.margot")
	if _, err := Pack(sourceProject, "brand", archivePath, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	targetProject := filepath.Join(parent, "target")
	writeTheme(t, targetProject, "brand")
	writeThemeFile(t, filepath.Join(targetProject, "themes", "brand", "keep.txt"), "keep")
	_, err := Import(targetProject, archivePath, "", "0.1.0")
	if err == nil || !strings.Contains(err.Error(), "theme already exists") {
		t.Fatalf("expected target conflict, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetProject, "themes", "brand", "keep.txt")); err != nil {
		t.Fatalf("existing theme changed: %v", err)
	}
}

func TestCompatibleVersion(t *testing.T) {
	for _, test := range []struct {
		current string
		minimum string
		want    bool
	}{
		{current: "0.2.0", minimum: "0.1.0", want: true},
		{current: "0.1.0", minimum: "0.1.0", want: true},
		{current: "0.1.0", minimum: "0.2.0", want: false},
		{current: "0.0.0-dev", minimum: "0.2.0", want: true},
	} {
		if got := compatibleVersion(test.current, test.minimum); got != test.want {
			t.Fatalf("compatibleVersion(%q, %q) = %t, want %t", test.current, test.minimum, got, test.want)
		}
	}
}

func writeTheme(t *testing.T, projectRoot, name string) {
	t.Helper()
	writeThemeFile(t, filepath.Join(projectRoot, "themes", name, "theme.yaml"), "name: "+name+"\nversion: 1.2.0\n")
	writeThemeFile(t, filepath.Join(projectRoot, "themes", name, "layouts", "default.html"), "{{ .Deck.Title }}")
}

func writeThemeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCustomArchive(t *testing.T, path string, manifest Manifest, files map[string]string) {
	t.Helper()
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	manifestFiles := []archiveFile{}
	for name, content := range files {
		manifestFiles = append(manifestFiles, archiveFile{name: name, data: []byte(content)})
	}
	if manifest.PayloadSHA256 == "" {
		manifest.PayloadSHA256 = payloadChecksum(manifestFiles)
	}
	if err := writeArchiveMember(writer, ManifestName, mustManifest(t, manifest)); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := writeArchiveMember(writer, name, []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustManifest(t *testing.T, manifest Manifest) []byte {
	t.Helper()
	data := "format_version: " + "1" + "\n" +
		"theme_name: " + manifest.ThemeName + "\n" +
		"theme_version: " + manifest.ThemeVersion + "\n" +
		"min_margo_version: " + manifest.MinMargo + "\n" +
		"payload_sha256: " + manifest.PayloadSHA256 + "\n"
	return []byte(data)
}

func writeArchiveMember(writer *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o644)
	w, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
