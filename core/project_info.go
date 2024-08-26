package core

import (
	"fmt"
	"os"
	"path/filepath"
)

type ProjectInfo struct {
	Name       string
	CWD        string
	TempDir    string
	TarFile    string
	ExtractDir string
	TxtarFile  string
}

func (mp *ManifestProcessor) setupProjectInfo() (ProjectInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectInfo{}, fmt.Errorf("error getting current working directory: %w", err)
	}

	projectName := filepath.Base(cwd)
	tempDir := filepath.Join(cwd, fmt.Sprintf("tmp_%s", projectName))
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return ProjectInfo{}, fmt.Errorf("error creating temp directory: %w", err)
	}

	mp.logger.V(1).Info("Created temporary directory", "path", tempDir)

	info := ProjectInfo{
		Name:       projectName,
		CWD:        cwd,
		TempDir:    tempDir,
		TarFile:    filepath.Join(tempDir, fmt.Sprintf("%s.tar", projectName)),
		ExtractDir: filepath.Join(tempDir, projectName),
		TxtarFile:  fmt.Sprintf("%s.txtar", projectName),
	}

	return info, nil
}
