package core

import (
	"fmt"
)

func (mg *ManifestGenerator) Generate(force bool, manifestFile string) (bool, error) {
	mg.logger.V(1).Info("Generating manifest")

	currentFiles, err := mg.walker.GetCurrentFiles()
	if err != nil {
		return false, fmt.Errorf("error getting current files: %w", err)
	}

	var manifest Manifest
	isNewManifest := true

	if !force {
		manifest, err = mg.reader.ReadManifest(manifestFile)
		if err != nil {
			return false, fmt.Errorf("error reading manifest: %w", err)
		}
		isNewManifest = len(manifest.FileList) == 0
	}

	if force || isNewManifest {
		manifest = Manifest{FileList: make(map[string]bool)}
		for file := range currentFiles {
			manifest.FileList[file] = true
		}
	} else {
		updatedManifest := mg.updater.UpdateManifest(manifest, currentFiles)
		manifest = updatedManifest
	}

	if err := mg.writer.WriteManifest(manifest, manifestFile); err != nil {
		return false, fmt.Errorf("error writing manifest: %w", err)
	}

	return isNewManifest || force, nil
}
