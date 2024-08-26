package core

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
)

type ManifestProcessor struct {
	logger       logr.Logger
	debug        bool
	manifestFile string
}

func NewManifestProcessor(logger logr.Logger, debug bool, manifestFile string) *ManifestProcessor {
	return &ManifestProcessor{
		logger:       logger,
		debug:        debug,
		manifestFile: manifestFile,
	}
}

func (mp *ManifestProcessor) Process() error {
	mp.logger.V(1).Info("Processing manifest")

	manifest, err := mp.readManifest()
	if err != nil {
		return err
	}

	projectInfo, err := mp.setupProjectInfo()
	if err != nil {
		return err
	}

	if !mp.debug {
		defer os.RemoveAll(projectInfo.TempDir)
	}

	if err := mp.processTarArchive(manifest, projectInfo); err != nil {
		return err
	}

	if err := mp.processTxtarArchive(projectInfo); err != nil {
		return err
	}

	if mp.debug {
		mp.logger.Info("Debug mode: Temporary directory kept for inspection", "path", projectInfo.TempDir)
	}

	mp.logger.Info("Manifest processed successfully")
	return nil
}

func (mp *ManifestProcessor) readManifest() (Manifest, error) {
	mp.logger.V(1).Info("Reading manifest file", "path", mp.manifestFile)

	manifestData, err := os.ReadFile(mp.manifestFile)
	if err != nil {
		return Manifest{}, fmt.Errorf("error reading manifest: %w", err)
	}

	var manifest Manifest
	err = yaml.Unmarshal(manifestData, &manifest)
	if err != nil {
		return Manifest{}, fmt.Errorf("error unmarshaling manifest: %w", err)
	}

	mp.logger.V(1).Info("Manifest read successfully", "fileCount", len(manifest.FileList))
	return manifest, nil
}
