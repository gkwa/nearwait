package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

func normalizePathForComparison(path string) (string, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(absPath)

	cleanPath = strings.TrimPrefix(cleanPath, "/private")

	return cleanPath, nil
}

func (mg *ManifestGenerator) CommentItem(manifestFile, item string) error {
	manifest, err := mg.ReadManifest(manifestFile)
	if err != nil {
		return err
	}

	normalizedItem, err := normalizePathForComparison(item)
	if err != nil {
		return err
	}
	if _, exists := manifest.FileList[normalizedItem]; exists {
		manifest.FileList[normalizedItem] = true
		return mg.WriteManifest(manifest, manifestFile)
	}

	return fmt.Errorf("item not found in manifest")
}

func (mg *ManifestGenerator) UncommentItem(manifestFile, item string) error {
	manifest, err := mg.ReadManifest(manifestFile)
	if err != nil {
		return err
	}

	normalizedItem, err := normalizePathForComparison(item)
	if err != nil {
		return err
	}
	if _, exists := manifest.FileList[normalizedItem]; exists {
		manifest.FileList[normalizedItem] = false
		return mg.WriteManifest(manifest, manifestFile)
	}

	return fmt.Errorf("item not found in manifest")
}

func (mg *ManifestGenerator) GetItemStatus(manifestFile, item string) (string, error) {
	manifest, err := mg.ReadManifest(manifestFile)
	if err != nil {
		return "", err
	}

	normalizedItem, err := normalizePathForComparison(item)
	if err != nil {
		return "", err
	}
	if isCommented, exists := manifest.FileList[normalizedItem]; exists {
		if isCommented {
			return "disabled", nil
		}
		return "enabled", nil
	}

	return "not in list", nil
}
