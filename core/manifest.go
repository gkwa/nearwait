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
	logger      logr.Logger
	excludeDirs map[string]bool
	reader      ManifestReader
	writer      ManifestWriter
	updater     ManifestUpdater
	walker      FileSystemWalker
	fsys        fs.FS
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
		fsys: nil,
	}
	mg.reader = mg
	mg.writer = mg
	mg.updater = mg
	mg.walker = mg
	return mg
}

func (mg *ManifestGenerator) WithFS(fsys fs.FS) *ManifestGenerator {
	mg.fsys = fsys
	return mg
}

func (mg *ManifestGenerator) isExcluded(path string) bool {
	mg.logger.V(1).Info("Checking if path is excluded", "path", path)
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")
	mg.logger.V(1).Info("Split path into parts", "parts", parts)
	for _, part := range parts {
		mg.logger.V(1).Info("Checking part", "part", part)
		if mg.excludeDirs[part] {
			mg.logger.V(1).Info("Part has been excluded", "part", part)
			return true
		}
		mg.logger.V(1).Info("Part is not excluded", "part", part)
	}
	mg.logger.V(1).Info("Path is not excluded", "path", path)
	return false
}
