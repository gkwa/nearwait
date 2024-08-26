package core

import (
	"fmt"
	"os"
)

func (mp *ManifestProcessor) processTarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	if err := mp.createTarArchive(manifest.FileList, projectInfo.TarFile, projectInfo.CWD); err != nil {
		return fmt.Errorf("error creating tar archive: %w", err)
	}

	mp.logger.V(1).Info("Created tar archive", "path", projectInfo.TarFile)

	if err := os.MkdirAll(projectInfo.ExtractDir, 0o755); err != nil {
		return fmt.Errorf("error creating extraction directory: %w", err)
	}

	mp.logger.V(1).Info("Created extraction directory", "path", projectInfo.ExtractDir)

	if err := mp.extractTarArchive(projectInfo.TarFile, projectInfo.ExtractDir); err != nil {
		return fmt.Errorf("error extracting tar archive: %w", err)
	}

	mp.logger.V(1).Info("Extracted tar archive", "from", projectInfo.TarFile, "to", projectInfo.ExtractDir)

	return nil
}
