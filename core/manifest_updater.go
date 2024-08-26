package core

func (mg *ManifestGenerator) UpdateManifest(manifest Manifest, currentFiles map[string]bool) Manifest {
	updatedManifest := Manifest{FileList: make(map[string]bool)}

	for file := range currentFiles {
		normalizedFile, err := normalizePathForComparison(file)
		if err != nil {
			mg.logger.Error(err, "Failed to normalize path", "path", file)
			continue
		}
		if isCommented, exists := manifest.FileList[normalizedFile]; exists {
			updatedManifest.FileList[normalizedFile] = isCommented
		} else {
			updatedManifest.FileList[normalizedFile] = true
		}
	}

	return updatedManifest
}
