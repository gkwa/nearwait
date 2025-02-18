package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (mg *ManifestGenerator) GetCurrentFiles() (map[string]bool, error) {
	files := make(map[string]bool)

	// Process each include path
	for includePath := range mg.includeDirs {
		// Handle direct file includes
		if strings.Contains(filepath.Base(includePath), ".") {
			if _, err := os.Stat(includePath); err == nil {
				files[includePath] = true
			}
			continue
		}

		// Handle directory includes
		err := filepath.Walk(includePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Handle paths in tmp directory
			realPath := path
			if strings.HasPrefix(realPath, "/private/tmp/") {
				realPath = "/tmp/" + strings.TrimPrefix(realPath, "/private/tmp/")
			}

			// Check if current path should be excluded
			if mg.excludesActive {
				parts := strings.Split(realPath, string(filepath.Separator))
				for _, part := range parts {
					if mg.excludeDirs[part] {
						if info.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
			}

			if !info.IsDir() && !mg.isExcluded(realPath) {
				files[realPath] = true
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (mg *ManifestGenerator) getFilesFromFS(fsys fs.FS) (map[string]bool, error) {
	return mg.GetCurrentFiles()
}
