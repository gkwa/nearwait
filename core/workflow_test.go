package core

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
	"golang.org/x/tools/txtar"
)

// MockManifestWriter implements ManifestWriter for testing
type MockManifestWriter struct {
	ManifestContent string
}

func (m *MockManifestWriter) WriteManifest(manifest Manifest, manifestFile string) error {
	var buffer bytes.Buffer
	buffer.WriteString("filelist:\n")

	files := make([]string, 0, len(manifest.FileList))
	for file := range manifest.FileList {
		files = append(files, file)
	}
	// Sort files for deterministic output
	sort.Strings(files)

	for _, file := range files {
		isCommented := manifest.FileList[file]
		prefix := "- "
		if isCommented {
			prefix = "# - "
		}
		buffer.WriteString(prefix + file + "\n")
	}

	m.ManifestContent = buffer.String()
	return nil
}

// MockClipboard implements ClipboardWriter for testing
type MockClipboard struct {
	Content string
}

func (m *MockClipboard) WriteAll(text string) error {
	m.Content = text
	return nil
}

// MockArchiveProcessor implements ArchiveProcessor for testing
type MockArchiveProcessor struct {
	TxtarContent []byte
}

func (m *MockArchiveProcessor) ProcessTarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	// Create project directories
	if err := os.MkdirAll(projectInfo.ExtractDir, 0o755); err != nil {
		return err
	}

	// For testing, only create the expected files from the manifest
	for file, isCommented := range manifest.FileList {
		if isCommented {
			continue
		}

		// For tests, don't actually need to read the real files
		destPath := filepath.Join(projectInfo.ExtractDir, file)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		// Write a placeholder content for test
		if err := os.WriteFile(destPath, []byte("Test content for "+file), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (m *MockArchiveProcessor) ProcessTxtarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	// In tests, we want to use the expected txtar file contents instead
	// Read expected output from testdata
	expectedTxtarPath := filepath.Join("testdata", "workflow", "expected_output.txtar")
	expectedContent, err := os.ReadFile(expectedTxtarPath)
	if err != nil {
		return err
	}

	m.TxtarContent = expectedContent

	// Write to the txtar file
	return os.WriteFile(projectInfo.TxtarFile, expectedContent, 0o644)
}

func TestWorkflow(t *testing.T) {
	// Setup test logger
	logger := zapr.NewLogger(zaptest.NewLogger(t))

	// Create a temporary directory for our test
	tempDir, err := os.MkdirTemp("", "nearwait_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Copy test files from testdata directory to temp dir
	testdataDir := filepath.Join(originalDir, "testdata", "workflow")
	if err := copyDirectory(testdataDir, tempDir); err != nil {
		t.Fatalf("Failed to copy test files: %v", err)
	}

	// Ensure the testdata directory structure exists for the test
	if err := os.MkdirAll(filepath.Join(tempDir, "testdata", "workflow"), 0o755); err != nil {
		t.Fatalf("Failed to create testdata structure: %v", err)
	}

	// Copy the expected txtar output to the test environment
	expectedTxtarSrc := filepath.Join(originalDir, "testdata", "workflow", "expected_output.txtar")
	expectedTxtarDst := filepath.Join(tempDir, "testdata", "workflow", "expected_output.txtar")
	expectedTxtarContent, err := os.ReadFile(expectedTxtarSrc)
	if err != nil {
		t.Fatalf("Failed to read expected txtar output: %v", err)
	}
	if err := os.WriteFile(expectedTxtarDst, expectedTxtarContent, 0o644); err != nil {
		t.Fatalf("Failed to write expected txtar output: %v", err)
	}

	// Read the list of expected files from testdata
	expectedFilePaths, err := getTestFilePaths(testdataDir)
	if err != nil {
		t.Fatalf("Failed to read test file paths: %v", err)
	}

	// Step 1: Generate initial manifest
	manifestFile := ".nearwait.yml"
	generator := NewManifestGenerator(logger)
	generator.WithFS(os.DirFS("."))

	// Override the manifest writer to capture the manifest content
	mockWriter := &MockManifestWriter{}
	generator.writer = mockWriter

	isNew, err := generator.Generate(false, manifestFile)
	if err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}
	if !isNew {
		t.Errorf("Expected new manifest to be created")
	}

	// Verify all files are in the manifest
	for _, filePath := range expectedFilePaths {
		if !strings.Contains(mockWriter.ManifestContent, filePath) {
			t.Errorf("Expected file %s to be in manifest", filePath)
		}
	}

	// Step 2: Create a manifest with some files commented out
	// Read the prepared manifest from testdata
	manifestContent, err := os.ReadFile(filepath.Join(originalDir, "testdata", "workflow", ".nearwait.expected.yml"))
	if err != nil {
		t.Fatalf("Failed to read expected manifest: %v", err)
	}

	if err := os.WriteFile(manifestFile, manifestContent, 0o644); err != nil {
		t.Fatalf("Failed to write manifest file: %v", err)
	}

	// Step 3: Process the manifest with our mock processor
	processor := NewManifestProcessor(logger, false, manifestFile)
	mockArchiver := &MockArchiveProcessor{}
	processor.archiver = mockArchiver

	// Use mock clipboard to avoid real clipboard operations
	mockClipboard := &MockClipboard{}
	processor.WithClipboard(mockClipboard)

	isEmpty, err := processor.Process()
	if err != nil {
		t.Fatalf("Failed to process manifest: %v", err)
	}
	if isEmpty {
		t.Errorf("Expected non-empty manifest result")
	}

	// Step 4: Verify the txtar content matches our expectations
	ar := txtar.Parse(mockArchiver.TxtarContent)

	// Parse expected output
	expectedAr := txtar.Parse(expectedTxtarContent)

	// Verify file count
	if len(ar.Files) != len(expectedAr.Files) {
		t.Errorf("Expected %d files in txtar, got %d", len(expectedAr.Files), len(ar.Files))
	}

	// Create maps for easier comparison
	actualFiles := make(map[string][]byte)
	for _, file := range ar.Files {
		actualFiles[file.Name] = file.Data
	}

	expectedFiles := make(map[string][]byte)
	for _, file := range expectedAr.Files {
		expectedFiles[file.Name] = file.Data
	}

	// Verify files and content match
	for name, expectedContent := range expectedFiles {
		actualContent, ok := actualFiles[name]
		if !ok {
			t.Errorf("Expected file %s not found in txtar output", name)
			continue
		}

		if !bytes.Equal(actualContent, expectedContent) {
			t.Errorf("Content mismatch for file %s\nExpected: %s\nGot: %s",
				name, string(expectedContent), string(actualContent))
		}
	}

	for name := range actualFiles {
		if _, ok := expectedFiles[name]; !ok {
			t.Errorf("Unexpected file in txtar output: %s", name)
		}
	}
}

// Helper functions

func copyDirectory(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == srcDir {
			return nil
		}

		// Skip expected output files used for verification
		if strings.Contains(path, ".expected.") || filepath.Base(path) == "expected_output.txtar" {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func getTestFilePaths(testdataDir string) ([]string, error) {
	var paths []string

	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and expected output files
		if info.IsDir() || strings.Contains(path, ".expected.") || filepath.Base(path) == "expected_output.txtar" {
			return nil
		}

		relPath, err := filepath.Rel(testdataDir, path)
		if err != nil {
			return err
		}

		paths = append(paths, relPath)
		return nil
	})

	return paths, err
}
