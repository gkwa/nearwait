package core

import (
	"os"

	"github.com/atotto/clipboard"
	"github.com/go-logr/logr"
)

type ArchiveProcessor interface {
	ProcessTarArchive(manifest Manifest, projectInfo ProjectInfo) error
	ProcessTxtarArchive(manifest Manifest, projectInfo ProjectInfo) error
}

type ClipboardWriter interface {
	WriteAll(text string) error
}

type SystemClipboard struct{}

func (c *SystemClipboard) WriteAll(text string) error {
	return clipboard.WriteAll(text)
}

type ManifestProcessor struct {
	logger       logr.Logger
	debug        bool
	manifestFile string
	reader       ManifestReader
	archiver     ArchiveProcessor
	clipboard    ClipboardWriter
}

func NewManifestProcessor(logger logr.Logger, debug bool, manifestFile string) *ManifestProcessor {
	mp := &ManifestProcessor{
		logger:       logger,
		debug:        debug,
		manifestFile: manifestFile,
		clipboard:    &SystemClipboard{},
	}
	mp.reader = NewManifestGenerator(logger)
	mp.archiver = mp
	return mp
}

// For testing purposes
func (mp *ManifestProcessor) WithClipboard(clipboard ClipboardWriter) *ManifestProcessor {
	mp.clipboard = clipboard
	return mp
}

func (mp *ManifestProcessor) Process() (bool, error) {
	mp.logger.V(1).Info("Processing manifest")

	manifest, err := mp.reader.ReadManifest(mp.manifestFile)
	if err != nil {
		return false, err
	}

	// Check if there are any uncommented entries in the manifest
	hasUncommentedEntries := false
	for _, isCommented := range manifest.FileList {
		if !isCommented {
			hasUncommentedEntries = true
			break
		}
	}

	if !hasUncommentedEntries {
		return true, nil
	}

	projectInfo, err := mp.setupProjectInfo()
	if err != nil {
		return false, err
	}

	if !mp.debug {
		defer os.RemoveAll(projectInfo.TempDir)
	}

	if err := mp.archiver.ProcessTarArchive(manifest, projectInfo); err != nil {
		return false, err
	}

	if err := mp.archiver.ProcessTxtarArchive(manifest, projectInfo); err != nil {
		return false, err
	}

	if mp.debug {
		mp.logger.Info("Debug mode: Temporary directory kept for inspection", "path", projectInfo.TempDir)
	}

	// If there are uncommented entries, copy txtar content to clipboard
	txtarContent, err := os.ReadFile(projectInfo.TxtarFile)
	if err != nil {
		return false, err
	}

	// Skip clipboard in test environment
	if mp.clipboard != nil {
		if err := mp.clipboard.WriteAll(string(txtarContent)); err != nil {
			mp.logger.V(1).Info("Skipping clipboard: " + err.Error())
		} else {
			mp.logger.V(1).Info("Txtar content copied to clipboard")
		}
	}

	return false, nil
}
