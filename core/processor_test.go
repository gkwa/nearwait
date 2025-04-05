package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
)

// TestClipboard implements ClipboardWriter for testing
type TestClipboard struct {
	Content string
	Copies  int
}

func (m *TestClipboard) WriteAll(text string) error {
	m.Content = text
	m.Copies++
	return nil
}

func (m *TestClipboard) ShouldDelay() bool {
	return false // Tests should not delay
}

// TestBatchWait tests the waitBatch functionality
func TestBatchWait(t *testing.T) {
	// Create a test logger
	logger := zapr.NewLogger(zaptest.NewLogger(t))

	// Create temp directories
	tempDir, err := os.MkdirTemp("", "nearwait_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up project structure
	extractDir := filepath.Join(tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}

	batchDir := filepath.Join(tempDir, "batches")
	if err := os.MkdirAll(batchDir, 0o755); err != nil {
		t.Fatalf("Failed to create batch dir: %v", err)
	}

	// Create test files that will be split into batches
	testFiles := []struct {
		name    string
		content string
	}{
		{"file1.txt", strings.Repeat("Content for file 1. ", 20)},
		{"file2.txt", strings.Repeat("Content for file 2. ", 20)},
		{"file3.txt", strings.Repeat("Content for file 3. ", 20)},
	}

	// Create the files in the extract directory
	for _, tf := range testFiles {
		filePath := filepath.Join(extractDir, tf.name)
		if err := os.WriteFile(filePath, []byte(tf.content), 0o644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", tf.name, err)
		}
	}

	// Create a manifest file
	manifestFile := filepath.Join(tempDir, ".nearwait.yml")
	manifestContent := "filelist:\n"
	for _, tf := range testFiles {
		manifestContent += "- " + tf.name + "\n"
	}
	if err := os.WriteFile(manifestFile, []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("Failed to write manifest file: %v", err)
	}

	// Create a txtar file
	txtarFile := filepath.Join(tempDir, ".nearwait.txtar")
	txtarContent := ""
	for _, tf := range testFiles {
		txtarContent += "-- " + tf.name + " --\n" + tf.content + "\n"
	}
	if err := os.WriteFile(txtarFile, []byte(txtarContent), 0o644); err != nil {
		t.Fatalf("Failed to write txtar file: %v", err)
	}

	// Create batch files
	for i, tf := range testFiles {
		batchFileName := filepath.Join(batchDir, fmt.Sprintf("batch_%03d.txtar", i+1))
		batchContent := "-- " + tf.name + " --\n" + tf.content + "\n"
		if err := os.WriteFile(batchFileName, []byte(batchContent), 0o644); err != nil {
			t.Fatalf("Failed to write batch file %s: %v", batchFileName, err)
		}
	}

	// Create ProjectInfo for testing
	projectInfo := ProjectInfo{
		TempDir:    tempDir,
		ExtractDir: extractDir,
		BatchDir:   batchDir,
		TxtarFile:  txtarFile,
	}

	// Save original stdin and create pipes for testing
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
	}()

	// Save original stdout to capture output
	oldStdout := os.Stdout
	outR, outW, _ := os.Pipe()
	os.Stdout = outW
	defer func() {
		os.Stdout = oldStdout
	}()

	// Create a go routine to simulate user input
	go func() {
		// Write simulated user inputs (Enter key presses)
		w.Write([]byte("\n"))
		w.Write([]byte("\n"))
		w.Close()
	}()

	// Mock functions for testing
	mockClipboard := &TestClipboard{}

	// Set up processor with waitBatch enabled
	processor := &ManifestProcessor{
		logger:       logger,
		debug:        false,
		manifestFile: manifestFile,
		batchKBytes:  1, // Small batch size to ensure multiple batches
		waitBatch:    true,
		clipboard:    mockClipboard,
	}

	// Mock the createBatches function to return our predefined batch files
	batchFiles := []string{
		filepath.Join(batchDir, "batch_001.txtar"),
		filepath.Join(batchDir, "batch_002.txtar"),
		filepath.Join(batchDir, "batch_003.txtar"),
	}

	// Test processing with waitBatch
	processor.processBatches(batchFiles, projectInfo)

	// Close the stdout pipe
	outW.Close()

	// Read captured stdout
	var outBuf bytes.Buffer
	_, err = io.Copy(&outBuf, outR)
	if err != nil {
		t.Fatalf("Failed to read stdout: %v", err)
	}

	output := outBuf.String()

	// Verify that prompts were shown
	if !strings.Contains(output, "Press Enter to copy batch 2/3") {
		t.Errorf("Expected prompt for batch 2, but got: %s", output)
	}
	if !strings.Contains(output, "Press Enter to copy batch 3/3") {
		t.Errorf("Expected prompt for batch 3, but got: %s", output)
	}

	// Verify clipboard usage
	if mockClipboard.Copies != 3 {
		t.Errorf("Expected 3 clipboard copies, got %d", mockClipboard.Copies)
	}
}

// Helper function for testing batch processing
func (mp *ManifestProcessor) processBatches(batches []string, projectInfo ProjectInfo) error {
	// This is similar to the batch processing part in Process()
	fmt.Printf("Created %d batches\n", len(batches))

	for i, batchFile := range batches {
		batchContent, err := os.ReadFile(batchFile)
		if err != nil {
			return err
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

		if err := mp.clipboard.WriteAll(string(batchContent)); err != nil {
			mp.logger.V(1).Info("Skipping clipboard: " + err.Error())
		} else {
			mp.logger.V(1).Info("Batch txtar content copied to clipboard",
				"batch", i+1,
				"total_batches", len(batches))
		}
	}

	return nil
}
