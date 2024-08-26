package core

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func (mg *ManifestGenerator) GetCurrentFiles() (map[string]bool, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for excludeDir := range mg.excludeDirs {
			if strings.Contains(path, excludeDir) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !d.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			normalizedPath, err := normalizePathForComparison(absPath)
			if err != nil {
				return err
			}
			files[normalizedPath] = true
		}

		return nil
	})
	return files, err
}
