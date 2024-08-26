package core

import (
	"fmt"
	"os"
)

func (mp *ManifestProcessor) ProcessTarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	var fileList []string
	for file, isCommented := range manifest.FileList {
		if !isCommented {
			fileList = append(fileList, file)
		}
	}

	if err := mp.createTarArchive(fileList, projectInfo.TarFile, projectInfo.CWD); err != nil {
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
