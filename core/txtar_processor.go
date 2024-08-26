package core

import (
	"fmt"
	"os"
)

func (mp *ManifestProcessor) ProcessTxtarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	var uncommentedFiles []string
	for file, isCommented := range manifest.FileList {
		if !isCommented {
			uncommentedFiles = append(uncommentedFiles, file)
		}
	}

	if len(uncommentedFiles) == 0 {
		if err := os.Remove(projectInfo.TxtarFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error deleting empty txtar archive: %w", err)
		}
		mp.logger.V(1).Info("No uncommented files, txtar archive not created or deleted if existed")
		return nil
	}

	txtarContent, err := mp.createTxtarArchive(projectInfo.ExtractDir)
	if err != nil {
		return fmt.Errorf("error creating txtar archive: %w", err)
	}

	if err := os.WriteFile(projectInfo.TxtarFile, txtarContent, 0o644); err != nil {
		return fmt.Errorf("error writing txtar archive: %w", err)
	}

	mp.logger.V(1).Info("Created txtar archive", "path", projectInfo.TxtarFile)

	return nil
}
