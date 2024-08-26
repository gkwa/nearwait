package core

import (
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
