package core

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mitchellh/go-homedir"
)

type Manifest struct {
	FileList map[string]bool
}

type ManifestReader interface {
	ReadManifest(manifestFile string) (Manifest, error)
}

type ManifestWriter interface {
	WriteManifest(manifest Manifest, manifestFile string) error
}

type ManifestUpdater interface {
	UpdateManifest(manifest Manifest, currentFiles map[string]bool) Manifest
}

type FileSystemWalker interface {
	GetCurrentFiles() (map[string]bool, error)
}

type ManifestGenerator struct {
	logger      logr.Logger
	excludeDirs map[string]bool
	reader      ManifestReader
	writer      ManifestWriter
	updater     ManifestUpdater
	walker      FileSystemWalker
}

func NewManifestGenerator(logger logr.Logger) *ManifestGenerator {
	mg := &ManifestGenerator{
		logger: logger,
		excludeDirs: map[string]bool{
			"__pycache__":   true,
			".git":          true,
			".nearwait.yml": true,
			".pytest_cache": true,
			".ruff_cache":   true,
			".terraform":    true,
			".timestamps":   true,
			".tox":          true,
			".venv":         true,
			"node_modules":  true,
			"target/debug":  true,
		},
	}
	mg.reader = mg
	mg.writer = mg
	mg.updater = mg
	mg.walker = mg
	return mg
}

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

func (mg *ManifestGenerator) WriteManifest(manifest Manifest, manifestFile string) error {
	file, err := os.Create(manifestFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString("filelist:\n")
	if err != nil {
		return err
	}

	var sortedFiles []string
	for file := range manifest.FileList {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Strings(sortedFiles)

	for _, file := range sortedFiles {
		isCommented := manifest.FileList[file]
		prefix := "- "
		if isCommented {
			prefix = "# - "
		}
		_, err = writer.WriteString(fmt.Sprintf("%s%s\n", prefix, file))
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

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

func (mg *ManifestGenerator) GetCurrentFiles() (map[string]bool, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for excludeDir := range mg.excludeDirs {
			if strings.Contains(path, excludeDir) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !d.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			normalizedPath, err := normalizePathForComparison(absPath)
			if err != nil {
				return err
			}
			files[normalizedPath] = true
		}

		return nil
	})
	return files, err
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
