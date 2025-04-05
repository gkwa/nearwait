package core

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func (mg *ManifestGenerator) GetCurrentFiles() (map[string]bool, error) {
	files := make(map[string]bool)

	// If no filesystem is provided, return an empty map
	if mg.fsys == nil {
		return files, nil
	}

	// If no includes are specified, walk the entire filesystem
	if len(mg.includeDirs) == 0 {
		err := fs.WalkDir(mg.fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip if it's a directory
			if d.IsDir() {
				if mg.isExcluded(path) {
					return fs.SkipDir
				}
				return nil
			}

			if !mg.isExcluded(path) {
				files[path] = true
			}
			return nil
		})
		return files, err
	}

	// Process each include path
	for includePath := range mg.includeDirs {
		// Handle direct file includes
		if strings.Contains(filepath.Base(includePath), ".") {
			info, err := fs.Stat(mg.fsys, includePath)
			if err == nil && !info.IsDir() {
				files[includePath] = true
			}
			continue
		}

		// Handle directory includes
		err := fs.WalkDir(mg.fsys, includePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip if it's a directory
			if d.IsDir() {
				if mg.isExcluded(path) {
					return fs.SkipDir
				}
				return nil
			}

			if !mg.isExcluded(path) {
				files[path] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}
