package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jjanuszczak/margo/internal/config"
)

type Root struct {
	Dir        string
	ConfigPath string
}

func Discover(startDir string) (Root, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return Root{}, fmt.Errorf("resolve working directory: %w", err)
	}

	configPath := filepath.Join(dir, config.DefaultFilename)
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return Root{}, fmt.Errorf("no %s found in %s", config.DefaultFilename, dir)
		}

		return Root{}, fmt.Errorf("stat config file %q: %w", configPath, err)
	}

	return Root{
		Dir:        dir,
		ConfigPath: configPath,
	}, nil
}
