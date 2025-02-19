package core

import (
	"os"
	"path/filepath"

	"golang.org/x/tools/txtar"
)

func (mp *ManifestProcessor) createTxtarArchive(extractDir string) ([]byte, error) {
	mp.logger.V(1).Info("Creating txtar archive", "from", extractDir)

	var ar txtar.Archive

	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip the .tar file
		if filepath.Ext(path) == ".tar" {
			return nil
		}

		relPath, err := filepath.Rel(extractDir, path)
		if err != nil {
			return err
		}

		mp.logger.V(1).Info("Adding file to txtar", "file", relPath)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		ar.Files = append(ar.Files, txtar.File{
			Name: relPath,
			Data: content,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return txtar.Format(&ar), nil
}
