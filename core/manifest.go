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
		mg.logger.V(1).Info("Added include directory", "dir", cleanDir)
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

	// First check excludes if they are active
	if mg.excludesActive {
		parts := strings.Split(cleanPath, string(filepath.Separator))
		for _, part := range parts {
			if mg.excludeDirs[part] {
				return true
			}
		}
	}

	// Then check includes if specified
	if len(mg.includeDirs) > 0 {
		// Check if this is a directory entry
		isDir := !strings.Contains(cleanPath, ".")

		for dir := range mg.includeDirs {
			// If it's the include dir itself or a file under it
			if cleanPath == dir || strings.HasPrefix(cleanPath, dir+string(filepath.Separator)) {
				return false
			}

			// If it's a directory that might contain included paths
			if isDir && strings.HasPrefix(dir, cleanPath+string(filepath.Separator)) {
				return false
			}
		}

		return true
	}

	return false
}
