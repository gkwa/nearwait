package core

import (
	"bufio"
	"os"
	"strings"
)

func (mg *ManifestGenerator) ReadManifest(manifestFile string) (Manifest, error) {
	manifest := Manifest{FileList: make(map[string]bool)}

	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		return manifest, nil
	}

	file, err := os.Open(manifestFile)
	if err != nil {
		return manifest, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# - ") {
			path := strings.TrimPrefix(line, "# - ")
			normalizedPath, err := normalizePathForComparison(path)
			if err != nil {
				return manifest, err
			}
			manifest.FileList[normalizedPath] = true
		} else if strings.HasPrefix(line, "- ") {
			path := strings.TrimPrefix(line, "- ")
			normalizedPath, err := normalizePathForComparison(path)
			if err != nil {
				return manifest, err
			}
			manifest.FileList[normalizedPath] = false
		}
	}

	return manifest, scanner.Err()
}
