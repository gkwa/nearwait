package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ProjectInfo struct {
	Name       string
	CWD        string
	TempDir    string
	TarFile    string
	ExtractDir string
	TxtarFile  string
	BatchDir   string
}

func (mp *ManifestProcessor) setupProjectInfo() (ProjectInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectInfo{}, fmt.Errorf("error getting current working directory: %w", err)
	}

	projectName := filepath.Base(cwd)
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("nearwait_%s", projectName))
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return ProjectInfo{}, fmt.Errorf("error creating temp directory: %w", err)
	}

	mp.logger.V(1).Info("Created temporary directory", "path", tempDir)

	manifestBasename := filepath.Base(mp.manifestFile)
	manifestBasename = strings.TrimSuffix(manifestBasename, filepath.Ext(manifestBasename))
	txtarFilename := fmt.Sprintf("%s.txtar", manifestBasename)

	batchDir := filepath.Join(tempDir, "batches")
	if mp.batchSize > 0 {
		if err := os.MkdirAll(batchDir, 0o755); err != nil {
			return ProjectInfo{}, fmt.Errorf("error creating batch directory: %w", err)
		}
	}

	info := ProjectInfo{
		Name:       projectName,
		CWD:        cwd,
		TempDir:    tempDir,
		TarFile:    filepath.Join(tempDir, fmt.Sprintf("%s.tar", projectName)),
		ExtractDir: filepath.Join(tempDir, projectName),
		TxtarFile:  filepath.Join(filepath.Dir(mp.manifestFile), txtarFilename),
		BatchDir:   batchDir,
	}

	return info, nil
}
