package core

import (
	"io/fs"
)

func (mg *ManifestGenerator) GetCurrentFiles() (map[string]bool, error) {
	return mg.getFilesFromFS(mg.fsys)
}

func (mg *ManifestGenerator) getFilesFromFS(fsys fs.FS) (map[string]bool, error) {
	files := make(map[string]bool)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if mg.isExcluded(path) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			files[path] = true
		}

		return nil
	})
	return files, err
}
