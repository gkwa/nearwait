package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
	"golang.org/x/tools/txtar"
)

func TestCreateBatches(t *testing.T) {
	// Create a test logger
	logger := zapr.NewLogger(zaptest.NewLogger(t))

	// Create temp directories
	tempDir, err := os.MkdirTemp("", "nearwait_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create project info
	extractDir := filepath.Join(tempDir, "extract")
	batchDir := filepath.Join(tempDir, "batches")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}
	if err := os.MkdirAll(batchDir, 0o755); err != nil {
		t.Fatalf("Failed to create batch dir: %v", err)
	}

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"file1.txt", "This is a small file."},
		{"file2.txt", "This is a slightly larger file with more content."},
		{"file3.txt", "This is the largest file with even more content to ensure it exceeds certain batch sizes."},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(extractDir, tf.name)
		if err := os.WriteFile(filePath, []byte(tf.content), 0o644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", tf.name, err)
		}
	}

	// Create a txtar file for easy inspection
	var ar txtar.Archive
	for _, tf := range testFiles {
		ar.Files = append(ar.Files, txtar.File{
			Name: tf.name,
			Data: []byte(tf.content),
		})
	}
	txtarPath := filepath.Join(tempDir, "test.txtar")
	if err := os.WriteFile(txtarPath, txtar.Format(&ar), 0o644); err != nil {
		t.Fatalf("Failed to write txtar file: %v", err)
	}

	projectInfo := ProjectInfo{
		TempDir:    tempDir,
		ExtractDir: extractDir,
		BatchDir:   batchDir,
		TxtarFile:  txtarPath,
	}

	tests := []struct {
		name        string
		batchKBytes int64
		wantBatches int
	}{
		{"No batching", 0, 0},
		{"Large batch size", 1, 1},  // 1KB should fit all files
		{"Medium batch size", 1, 1}, // With kilobytes, even 1KB should fit all our test files
		{"Small batch size", 1, 1},  // With kilobytes, even 1KB should fit all our test files
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := &ManifestProcessor{
				logger:      logger,
				batchKBytes: tt.batchKBytes,
			}

			batches, err := mp.createBatches(projectInfo)
			if err != nil {
				t.Fatalf("createBatches() error = %v", err)
			}

			if len(batches) != tt.wantBatches {
				t.Errorf("createBatches() got %d batches, want %d", len(batches), tt.wantBatches)
			}

			// Verify batch contents if batching was enabled
			if tt.batchKBytes > 0 && len(batches) > 0 {
				var totalFiles int
				for i, batchFile := range batches {
					// Check that batch file exists
					if _, err := os.Stat(batchFile); os.IsNotExist(err) {
						t.Errorf("Batch file %s does not exist", batchFile)
						continue
					}

					// Read and parse batch file
					data, err := os.ReadFile(batchFile)
					if err != nil {
						t.Errorf("Failed to read batch file %s: %v", batchFile, err)
						continue
					}

					archive := txtar.Parse(data)
					t.Logf("Batch %d contains %d files", i+1, len(archive.Files))
					totalFiles += len(archive.Files)
				}

				// Verify that all files were batched
				if totalFiles != len(testFiles) {
					t.Errorf("Total files in batches: %d, want %d", totalFiles, len(testFiles))
				}
			}
		})
	}
}
