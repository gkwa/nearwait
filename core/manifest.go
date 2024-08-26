package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
)

type Manifest struct {
	FileList []string `yaml:"filelist"`
}

type ManifestGenerator struct {
	logger      logr.Logger
	excludeDirs map[string]bool
}

func NewManifestGenerator(logger logr.Logger) *ManifestGenerator {
	return &ManifestGenerator{
		logger: logger,
		excludeDirs: map[string]bool{
			"__pycache__":             true,
			".git":                    true,
			".tox":                    true,
			".ruff_cache":             true,
			".pytest_cache":           true,
			".terraform":              true,
			".timestamps":             true,
			".venv":                   true,
			"gpt_instructions_XXYYBB": true,
			"node_modules":            true,
			"target/debug":            true,
		},
	}
}

func (mg *ManifestGenerator) Generate(force bool, manifestFile string) error {
	mg.logger.V(1).Info("Generating manifest")

	manifest := Manifest{}
	err := mg.walkDirectory(&manifest)
	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	if err := mg.writeManifest(manifest, force, manifestFile); err != nil {
		return err
	}

	mg.logger.Info("Manifest generated successfully")
	return nil
}

func (mg *ManifestGenerator) walkDirectory(manifest *Manifest) error {
	return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
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
			manifest.FileList = append(manifest.FileList, absPath)
		}

		return nil
	})
}

func (mg *ManifestGenerator) writeManifest(manifest Manifest, force bool, manifestFile string) error {
	if _, err := os.Stat(manifestFile); err == nil && !force {
		mg.logger.Info("Manifest file already exists and --force flag not set")
		return nil
	}

	yamlData := "filelist:\n"
	for _, file := range manifest.FileList {
		yamlData += fmt.Sprintf("#    - %s\n", file)
	}

	err := os.WriteFile(manifestFile, []byte(yamlData), 0o644)
	if err != nil {
		return fmt.Errorf("error writing manifest: %w", err)
	}

	return nil
}
