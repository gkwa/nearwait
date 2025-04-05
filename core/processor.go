package core

import (
	"bufio"
	"fmt"
	"os"
	"time"

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
	batchKBytes  int64
	waitBatch    bool
	reader       ManifestReader
	archiver     ArchiveProcessor
	clipboard    ClipboardWriter
}

func NewManifestProcessor(logger logr.Logger, debug bool, manifestFile string) *ManifestProcessor {
	mp := &ManifestProcessor{
		logger:       logger,
		debug:        debug,
		manifestFile: manifestFile,
		batchKBytes:  0,
		waitBatch:    false,
		clipboard:    &SystemClipboard{},
	}
	mp.reader = NewManifestGenerator(logger)
	mp.archiver = mp
	return mp
}

// WithBatchKBytes sets the maximum size for each batch of files in kilobytes
func (mp *ManifestProcessor) WithBatchKBytes(batchKBytes int64) *ManifestProcessor {
	mp.batchKBytes = batchKBytes
	return mp
}

// WithWaitBatch sets whether to wait for user confirmation between batches
func (mp *ManifestProcessor) WithWaitBatch(waitBatch bool) *ManifestProcessor {
	mp.waitBatch = waitBatch
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

	// Skip clipboard in test environment
	if mp.clipboard != nil {
		if mp.batchKBytes <= 0 {
			// No batching, copy everything at once
			// Read txtar content
			txtarContent, err := os.ReadFile(projectInfo.TxtarFile)
			if err != nil {
				return false, err
			}

			// Add delay before writing to clipboard
			time.Sleep(1 * time.Second)
			if err := mp.clipboard.WriteAll(string(txtarContent)); err != nil {
				mp.logger.V(1).Info("Skipping clipboard: " + err.Error())
			} else {
				mp.logger.V(1).Info("Txtar content copied to clipboard")
			}
		} else {
			// Create batches and copy each batch separately
			batches, err := mp.createBatches(projectInfo)
			if err != nil {
				return false, err
			}

			// Output batch count to stdout
			fmt.Printf("Created %d batches\n", len(batches))

			// Copy all batches to clipboard in sequence, from first to last
			for i, batchFile := range batches {
				batchContent, err := os.ReadFile(batchFile)
				if err != nil {
					return false, err
				}

				// If this is not the first batch and waitBatch is enabled, prompt user
				if i > 0 && mp.waitBatch {
					fmt.Printf("Press Enter to copy batch %d/%d...", i+1, len(batches))
					reader := bufio.NewReader(os.Stdin)
					_, err = reader.ReadString('\n') // Wait for Enter key
					if err != nil {
						mp.logger.V(1).Info("Error reading user input", "error", err.Error())
					}
				}

				// Add delay before writing to clipboard
				time.Sleep(1 * time.Second)
				if err := mp.clipboard.WriteAll(string(batchContent)); err != nil {
					mp.logger.V(1).Info("Skipping clipboard: " + err.Error())
				} else {
					mp.logger.V(1).Info("Batch txtar content copied to clipboard",
						"batch", i+1,
						"total_batches", len(batches))
				}
			}

			// Log information about all batches
			mp.logger.V(1).Info("Created and copied batch files",
				"count", len(batches),
				"dir", projectInfo.BatchDir)
		}
	}

	return false, nil
}
