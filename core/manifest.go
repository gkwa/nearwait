package core

import (
	"io/fs"
	"path/filepath"
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
	logger         logr.Logger
	excludeDirs    map[string]bool
	includeDirs    map[string]bool
	reader         ManifestReader
	writer         ManifestWriter
	updater        ManifestUpdater
	walker         FileSystemWalker
	fsys           fs.FS
	excludesActive bool
}

func NewManifestGenerator(logger logr.Logger) *ManifestGenerator {
	mg := &ManifestGenerator{
		logger: logger,
		excludeDirs: map[string]bool{
			"__pycache__":       true,
			".git":              true,
			".nearwait.yml":     true,
			".pytest_cache":     true,
			".ruff_cache":       true,
			".terraform":        true,
			".terragrunt-cache": true,
			".timestamps":       true,
			".tox":              true,
			".venv":             true,
			"node_modules":      true,
			"target/debug":      true,
		},
		includeDirs:    make(map[string]bool),
		fsys:           nil,
		excludesActive: true,
	}
	mg.reader = mg
	mg.writer = mg
	mg.updater = mg
	mg.walker = mg
	return mg
}

func (mg *ManifestGenerator) DisableExcludes() {
	mg.excludesActive = false
}

func (mg *ManifestGenerator) WithIncludes(includes []string) *ManifestGenerator {
	for _, dir := range includes {
		cleanDir := filepath.Clean(dir)
		mg.includeDirs[cleanDir] = true
		mg.logger.V(1).Info("Added include path", "path", cleanDir)
	}
	return mg
}

func (mg *ManifestGenerator) WithFS(fsys fs.FS) *ManifestGenerator {
	mg.fsys = fsys
	return mg
}

func (mg *ManifestGenerator) isExcluded(path string) bool {
	if path == "." {
		return false
	}

	cleanPath := filepath.Clean(path)

	// Check if the path is in our includes
	if len(mg.includeDirs) > 0 {
		// First check exact matches
		if mg.includeDirs[cleanPath] {
			return false
		}

		// Then check if it's under any include directory
		for include := range mg.includeDirs {
			if !strings.Contains(filepath.Base(include), ".") { // if include is a directory
				if strings.HasPrefix(cleanPath, include+string(filepath.Separator)) {
					return false
				}
			}
		}
		return true
	}

	// Check excludes if no includes are specified
	if mg.excludesActive {
		parts := strings.Split(cleanPath, string(filepath.Separator))
		for _, part := range parts {
			if mg.excludeDirs[part] {
				return true
			}
		}
	}

	return false
}
