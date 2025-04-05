package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/txtar"
)

// FileInfo holds metadata about a file for batching purposes
type FileInfo struct {
	Path string
	Size int64
}

// createBatches creates multiple txtar files based on the batch size
func (mp *ManifestProcessor) createBatches(projectInfo ProjectInfo) ([]string, error) {
	// If batching is disabled, return
	if mp.batchSize <= 0 {
		return nil, nil
	}

	mp.logger.V(1).Info("Creating batched txtar archives",
		"batch_size", mp.batchSize,
		"extract_dir", projectInfo.ExtractDir)

	// Get all files and their sizes in the extraction directory
	var files []FileInfo
	err := filepath.Walk(projectInfo.ExtractDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(projectInfo.ExtractDir, path)
			if err != nil {
				return err
			}

			// Account for txtar overhead: each file adds a header line plus a newline
			// The txtar format adds: "-- filename --\n" + content + "\n"
			overhead := int64(len("-- "+relPath+" --\n") + 1) // +1 for the trailing newline

			files = append(files, FileInfo{
				Path: relPath,
				Size: info.Size() + overhead,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort files by size in descending order
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})

	// Create batches
	var batches [][]FileInfo
	var currentBatch []FileInfo
	var currentSize int64

	// Add each file to a batch
	for _, file := range files {
		// If file is larger than batch size, create its own batch
		if file.Size > mp.batchSize {
			batches = append(batches, []FileInfo{file})
			continue
		}

		// If adding this file would exceed batch size, start a new batch
		if currentSize+file.Size > mp.batchSize && len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = []FileInfo{file}
			currentSize = file.Size
		} else {
			// Add file to current batch
			currentBatch = append(currentBatch, file)
			currentSize += file.Size
		}
	}

	// Add the last batch if not empty
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	// Create a txtar file for each batch
	var batchFiles []string
	for i, batch := range batches {
		batchFileName := filepath.Join(projectInfo.BatchDir, fmt.Sprintf("batch_%03d.txtar", i+1))

		// Create a txtar archive for this batch
		var ar txtar.Archive
		for _, file := range batch {
			fullPath := filepath.Join(projectInfo.ExtractDir, file.Path)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, err
			}

			ar.Files = append(ar.Files, txtar.File{
				Name: file.Path,
				Data: content,
			})
		}

		// Write the batch to a file
		if err := os.WriteFile(batchFileName, txtar.Format(&ar), 0o644); err != nil {
			return nil, err
		}

		mp.logger.V(1).Info("Created batch txtar file",
			"batch", i+1,
			"file_count", len(batch),
			"path", batchFileName)

		batchFiles = append(batchFiles, batchFileName)
	}

	return batchFiles, nil
}
