package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const DefaultFilename = ".margoignore"

type Matcher struct {
	patterns []pattern
}

type pattern struct {
	raw       string
	value     string
	directory bool
	hasSlash  bool
}

func Load(projectRoot string) (Matcher, error) {
	path := filepath.Join(projectRoot, DefaultFilename)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Matcher{}, nil
		}
		return Matcher{}, err
	}
	defer file.Close()

	var patterns []pattern
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		directory := strings.HasSuffix(line, "/")
		line = strings.TrimSuffix(line, "/")
		line = filepath.ToSlash(filepath.Clean(line))
		if line == "." || line == "" {
			continue
		}

		patterns = append(patterns, pattern{
			raw:       scanner.Text(),
			value:     line,
			directory: directory,
			hasSlash:  strings.Contains(line, "/"),
		})
	}
	if err := scanner.Err(); err != nil {
		return Matcher{}, err
	}

	return Matcher{patterns: patterns}, nil
}

func (m Matcher) ShouldIgnore(rel string, isDir bool) bool {
	if rel == "" || rel == "." {
		return false
	}
	rel = filepath.ToSlash(filepath.Clean(rel))

	for _, p := range m.patterns {
		if p.matches(rel, isDir) {
			return true
		}
	}
	return false
}

func (p pattern) matches(rel string, isDir bool) bool {
	if p.directory {
		if !p.hasSlash {
			segments := strings.Split(rel, "/")
			for _, segment := range segments {
				if segment == p.value {
					return true
				}
			}
			return false
		}
		if rel == p.value {
			return true
		}
		return strings.HasPrefix(rel, p.value+"/")
	}

	if p.hasSlash {
		if ok, _ := filepath.Match(p.value, rel); ok {
			return true
		}
		return rel == p.value
	}

	base := filepath.Base(rel)
	if ok, _ := filepath.Match(p.value, base); ok {
		return true
	}
	return base == p.value
}
