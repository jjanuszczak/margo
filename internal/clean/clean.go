package clean

import (
	"fmt"
	"os"
	"path/filepath"
)

func Project(projectRoot string) error {
	distDir := filepath.Join(projectRoot, "dist")
	if _, err := os.Stat(distDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat dist directory %q: %w", distDir, err)
	}

	if err := os.RemoveAll(distDir); err != nil {
		return fmt.Errorf("remove dist directory %q: %w", distDir, err)
	}

	return nil
}
