package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/go-logr/logr"
)

type ArchiveProcessor interface {
	ProcessTarArchive(manifest Manifest, projectInfo ProjectInfo) error
	ProcessTxtarArchive(manifest Manifest, projectInfo ProjectInfo) error
}

type FileBatch struct {
	Files []string
	Size  int64
}

type ManifestProcessor struct {
	logger       logr.Logger
	debug        bool
	manifestFile string
	byteSize     int64
	reader       ManifestReader
	archiver     ArchiveProcessor
}

func NewManifestProcessor(logger logr.Logger, debug bool, manifestFile string) *ManifestProcessor {
	mp := &ManifestProcessor{
		logger:       logger,
		debug:        debug,
		manifestFile: manifestFile,
		byteSize:     0, // default to no batching
	}
	mp.reader = NewManifestGenerator(logger)
	mp.archiver = mp
	return mp
}

func (mp *ManifestProcessor) WithByteSize(byteSize int64) *ManifestProcessor {
	mp.byteSize = byteSize
	return mp
}

func (mp *ManifestProcessor) Process() (bool, error) {
	mp.logger.V(1).Info("Processing manifest")

	manifest, err := mp.reader.ReadManifest(mp.manifestFile)
	if err != nil {
		return false, err
	}

	// Check if there are any uncommented entries in the manifest
	var uncommentedFiles []string
	for file, isCommented := range manifest.FileList {
		if !isCommented {
			uncommentedFiles = append(uncommentedFiles, file)
		}
	}

	if len(uncommentedFiles) == 0 {
		return true, nil
	}

	// If byteSize is specified, process in batches
	if mp.byteSize > 0 {
		return mp.processBatches(uncommentedFiles)
	}

	// Otherwise process all files at once
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

	// Copy txtar content to clipboard
	txtarContent, err := os.ReadFile(projectInfo.TxtarFile)
	if err != nil {
		return false, err
	}

	if err := clipboard.WriteAll(string(txtarContent)); err != nil {
		return false, err
	}

	mp.logger.V(1).Info("Txtar content copied to clipboard")

	return false, nil
}

func (mp *ManifestProcessor) processBatches(uncommentedFiles []string) (bool, error) {
	// Create batches based on file sizes
	batches, err := mp.createBatches(uncommentedFiles)
	if err != nil {
		return false, err
	}

	mp.logger.Info(fmt.Sprintf("Created %d batches of files", len(batches)))

	// Process each batch
	for i, batch := range batches {
		mp.logger.Info(fmt.Sprintf("Processing batch %d/%d (%d files, %d bytes)",
			i+1, len(batches), len(batch.Files), batch.Size))

		// Log files in this batch
		for j, file := range batch.Files {
			mp.logger.V(1).Info(fmt.Sprintf("  Batch %d file %d: %s", i+1, j+1, file))
		}

		// Create a fresh project info for each batch with a unique temp directory
		projectInfo, err := mp.setupBatchProjectInfo(i + 1)
		if err != nil {
			return false, err
		}

		// Clean up after this batch unless debug is enabled
		if !mp.debug {
			defer os.RemoveAll(projectInfo.TempDir)
		}

		// Create a batch-specific manifest
		batchManifest := Manifest{FileList: make(map[string]bool)}
		for _, file := range batch.Files {
			batchManifest.FileList[file] = false
		}

		// Process this batch
		if err := mp.processBatch(batchManifest, projectInfo); err != nil {
			return false, err
		}

		// Wait for user to press enter before processing the next batch
		if i < len(batches)-1 {
			fmt.Printf("Batch %d/%d copied to clipboard. Press Enter to process next batch...", i+1, len(batches))
			fmt.Scanln()
		} else {
			fmt.Printf("Final batch %d/%d copied to clipboard.\n", i+1, len(batches))
		}
	}

	return false, nil
}

func (mp *ManifestProcessor) processBatch(manifest Manifest, projectInfo ProjectInfo) error {
	// Clear any existing txtar file before processing
	if _, err := os.Stat(projectInfo.TxtarFile); err == nil {
		if err := os.Remove(projectInfo.TxtarFile); err != nil {
			return fmt.Errorf("error removing existing txtar file: %w", err)
		}
	}

	if err := mp.archiver.ProcessTarArchive(manifest, projectInfo); err != nil {
		return fmt.Errorf("error processing tar archive: %w", err)
	}

	if err := mp.archiver.ProcessTxtarArchive(manifest, projectInfo); err != nil {
		return fmt.Errorf("error processing txtar archive: %w", err)
	}

	if mp.debug {
		mp.logger.Info("Debug mode: Temporary directory kept for inspection", "path", projectInfo.TempDir)
	}

	// Copy txtar content to clipboard
	txtarContent, err := os.ReadFile(projectInfo.TxtarFile)
	if err != nil {
		return fmt.Errorf("error reading txtar file: %w", err)
	}

	if err := clipboard.WriteAll(string(txtarContent)); err != nil {
		return fmt.Errorf("error copying to clipboard: %w", err)
	}

	mp.logger.V(1).Info("Batch txtar content copied to clipboard")
	return nil
}

func (mp *ManifestProcessor) createBatches(files []string) ([]FileBatch, error) {
	var batches []FileBatch
	var currentBatch FileBatch
	currentBatch.Files = []string{}
	currentBatch.Size = 0

	// First, check if any files exceed the byteSize limit
	var oversizedFiles []string
	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("error getting file size for %s: %w", file, err)
		}

		fileSize := fileInfo.Size()

		// If a single file is larger than byteSize, warn but include it in its own batch
		if fileSize > mp.byteSize {
			mp.logger.Info(fmt.Sprintf("Warning: File %s size (%d bytes) exceeds batch size limit (%d bytes)",
				file, fileSize, mp.byteSize))
			oversizedFiles = append(oversizedFiles, file)
		}
	}

	// Process normal-sized files
	for _, file := range files {
		// Skip oversized files; we'll handle them separately
		isOversized := false
		for _, oversizedFile := range oversizedFiles {
			if file == oversizedFile {
				isOversized = true
				break
			}
		}
		if isOversized {
			continue
		}

		// Get the file size
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("error getting file size: %w", err)
		}

		fileSize := fileInfo.Size()

		// If adding this file would exceed the batch size limit, create a new batch
		if currentBatch.Size+fileSize > mp.byteSize && len(currentBatch.Files) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = FileBatch{
				Files: []string{},
				Size:  0,
			}
		}

		// Add the file to the current batch
		currentBatch.Files = append(currentBatch.Files, file)
		currentBatch.Size += fileSize
		mp.logger.V(1).Info("Added file to batch", "file", file, "size", fileSize, "batchSize", currentBatch.Size)
	}

	// Add the last batch if it has any files
	if len(currentBatch.Files) > 0 {
		batches = append(batches, currentBatch)
	}

	// Now add each oversized file as its own batch
	for _, file := range oversizedFiles {
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("error getting file size: %w", err)
		}

		oversizedBatch := FileBatch{
			Files: []string{file},
			Size:  fileInfo.Size(),
		}
		batches = append(batches, oversizedBatch)
		mp.logger.V(1).Info("Added oversized file as its own batch", "file", file, "size", fileInfo.Size())
	}

	return batches, nil
}

// setupBatchProjectInfo creates a unique ProjectInfo for each batch
func (mp *ManifestProcessor) setupBatchProjectInfo(batchNum int) (ProjectInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectInfo{}, fmt.Errorf("error getting current working directory: %w", err)
	}

	projectName := filepath.Base(cwd)
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("nearwait_%s_batch%d", projectName, batchNum))

	// Ensure we have a clean temporary directory
	if _, err := os.Stat(tempDir); err == nil {
		if err := os.RemoveAll(tempDir); err != nil {
			return ProjectInfo{}, fmt.Errorf("error removing existing temp directory: %w", err)
		}
	}

	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return ProjectInfo{}, fmt.Errorf("error creating temp directory: %w", err)
	}

	mp.logger.V(1).Info("Created temporary directory", "path", tempDir)

	manifestBasename := filepath.Base(mp.manifestFile)
	manifestBasename = strings.TrimSuffix(manifestBasename, filepath.Ext(manifestBasename))
	txtarFilename := fmt.Sprintf("%s.txtar", manifestBasename)

	info := ProjectInfo{
		Name:       projectName,
		CWD:        cwd,
		TempDir:    tempDir,
		TarFile:    filepath.Join(tempDir, fmt.Sprintf("%s.tar", projectName)),
		ExtractDir: filepath.Join(tempDir, projectName),
		TxtarFile:  filepath.Join(filepath.Dir(mp.manifestFile), txtarFilename),
	}

	return info, nil
}
