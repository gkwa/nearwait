package core

import (
	"io/fs"
	"path/filepath"
)

func (mg *ManifestGenerator) GetCurrentFiles() (map[string]bool, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		mg.logger.V(1).Info("Encountered file before applying filters", "path", path)

		basename := filepath.Base(path)
		if _, exists := mg.excludeDirs[basename]; exists {
			mg.logger.V(1).Info("Exclusion filter applied", "path", path, "filter", basename)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
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
			mg.logger.V(1).Info("Added file to manifest", "path", normalizedPath)
		}

		return nil
	})
	return files, err
}
