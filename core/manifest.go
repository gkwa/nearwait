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

	manifest, err := mg.reader.ReadManifest(manifestFile)
	if err != nil {
		return false, fmt.Errorf("error reading manifest: %w", err)
	}

	isNewManifest := len(manifest.FileList) == 0
	updatedManifest := mg.updater.UpdateManifest(manifest, currentFiles)

	if err := mg.writer.WriteManifest(updatedManifest, manifestFile); err != nil {
		return false, fmt.Errorf("error writing manifest: %w", err)
	}

	return isNewManifest, nil
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
			manifest.FileList[strings.TrimPrefix(line, "# - ")] = true
		} else if strings.HasPrefix(line, "- ") {
			manifest.FileList[strings.TrimPrefix(line, "- ")] = false
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
		if isCommented, exists := manifest.FileList[file]; exists {
			updatedManifest.FileList[file] = isCommented
		} else {
			updatedManifest.FileList[file] = true
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
			files[absPath] = true
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

	if _, exists := manifest.FileList[item]; exists {
		manifest.FileList[item] = true
		return mg.WriteManifest(manifest, manifestFile)
	}

	return fmt.Errorf("item not found in manifest")
}

func (mg *ManifestGenerator) UncommentItem(manifestFile, item string) error {
	manifest, err := mg.ReadManifest(manifestFile)
	if err != nil {
		return err
	}

	if _, exists := manifest.FileList[item]; exists {
		manifest.FileList[item] = false
		return mg.WriteManifest(manifest, manifestFile)
	}

	return fmt.Errorf("item not found in manifest")
}

func (mg *ManifestGenerator) GetItemStatus(manifestFile, item string) (string, error) {
	manifest, err := mg.ReadManifest(manifestFile)
	if err != nil {
		return "", err
	}

	if isCommented, exists := manifest.FileList[item]; exists {
		if isCommented {
			return "disabled", nil
		}
		return "enabled", nil
	}

	return "not in list", nil
}
