package core

import (
	"fmt"
	"os"
)

func (mp *ManifestProcessor) processTxtarArchive(projectInfo ProjectInfo) error {
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
