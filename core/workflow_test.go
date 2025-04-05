package core

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
	"golang.org/x/tools/txtar"
)

// MockManifestWriter implements ManifestWriter and ManifestReader for testing
type MockManifestWriter struct {
	ManifestContent string
	ManifestData    Manifest
}

func (m *MockManifestWriter) WriteManifest(manifest Manifest, manifestFile string) error {
	m.ManifestData = manifest

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

func (m *MockManifestWriter) ReadManifest(manifestFile string) (Manifest, error) {
	return m.ManifestData, nil
}

// MockClipboard implements ClipboardWriter for testing
type MockClipboard struct {
	Content string
}

func (m *MockClipboard) WriteAll(text string) error {
	m.Content = text
	return nil
}

// Add this new method to implement the updated ClipboardWriter interface
func (m *MockClipboard) ShouldDelay() bool {
	return false // Tests should not have delays
}

// MockFileSystemWalker implements FileSystemWalker for testing
type MockFileSystemWalker struct {
	Files map[string]bool
}

func (m *MockFileSystemWalker) GetCurrentFiles() (map[string]bool, error) {
	return m.Files, nil
}

// MockArchiveProcessor implements ArchiveProcessor for testing
type MockArchiveProcessor struct {
	TxtarContent []byte
}

func (m *MockArchiveProcessor) ProcessTarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	// Create the extraction directory
	if err := os.MkdirAll(projectInfo.ExtractDir, 0o755); err != nil {
		return err
	}

	// Add some content to the extract directory so we'll have something in the txtar
	for file, isCommented := range manifest.FileList {
		if isCommented {
			continue
		}

		// Only create files that aren't commented out
		filePath := filepath.Join(projectInfo.ExtractDir, file)

		// Create the directory structure
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return err
		}

		// Write some test content
		if err := os.WriteFile(filePath, []byte("Mock content for "+file), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (m *MockArchiveProcessor) ProcessTxtarArchive(manifest Manifest, projectInfo ProjectInfo) error {
	// Create a txtar archive with the files in the extract directory
	var ar txtar.Archive

	// Add the files that aren't commented out
	for file, isCommented := range manifest.FileList {
		if isCommented {
			continue
		}

		ar.Files = append(ar.Files, txtar.File{
			Name: file,
			Data: []byte("Mock content for " + file),
		})
	}

	// Format the archive
	m.TxtarContent = txtar.Format(&ar)

	// Write to the txtar file
	return os.WriteFile(projectInfo.TxtarFile, m.TxtarContent, 0o644)
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

	// Create mock files for testing
	mockFiles := map[string]bool{
		"go.mod":            true,
		"main.go":           true,
		"internal/util.go":  true,
		"cmd/cli.go":        true,
		"README.md":         true,
		"testdata/test.txt": true,
	}

	// Create the necessary directory structure
	for file := range mockFiles {
		dir := filepath.Dir(file)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		// Create empty file
		if err := os.WriteFile(file, []byte("test content"), 0o644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// Step 1: Generate initial manifest with mocked file system walker
	manifestFile := ".nearwait.yml"
	generator := NewManifestGenerator(logger)
	generator.WithFS(os.DirFS("."))

	// Mock the file walker to return our predefined files
	mockWalker := &MockFileSystemWalker{Files: mockFiles}
	generator.walker = mockWalker

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

	// Create a manifest with some files commented out
	testManifest := Manifest{
		FileList: map[string]bool{
			"go.mod":            false,
			"main.go":           false,
			"internal/util.go":  false,
			"cmd/cli.go":        true, // commented out
			"README.md":         true, // commented out
			"testdata/test.txt": true, // commented out
		},
	}

	// Write test manifest content to file
	if err := mockWriter.WriteManifest(testManifest, manifestFile); err != nil {
		t.Fatalf("Failed to write manifest file: %v", err)
	}

	// Step 3: Process the manifest with our mock processor
	processor := NewManifestProcessor(logger, false, manifestFile)
	mockArchiver := &MockArchiveProcessor{}
	processor.archiver = mockArchiver
	processor.reader = mockWriter // Use our mock writer as reader too

	// Use mock clipboard to avoid real clipboard operations
	mockClipboard := &MockClipboard{}
	processor.WithClipboard(mockClipboard)

	// Create a basic txtarFile in the temp directory that will be read by the processor
	txtarFile := filepath.Join(tempDir, ".nearwait.txtar")
	dummyContent := []byte("-- go.mod --\nmodule test\n\ngo 1.20\n")
	if err := os.WriteFile(txtarFile, dummyContent, 0o644); err != nil {
		t.Fatalf("Failed to write dummy txtar file: %v", err)
	}

	isEmpty, err := processor.Process()
	if err != nil {
		t.Fatalf("Failed to process manifest: %v", err)
	}
	if isEmpty {
		t.Errorf("Expected non-empty manifest result")
	}

	// Verify the mock processor was called correctly and clipboard was used
	if mockClipboard.Content == "" {
		t.Errorf("Expected clipboard to be used")
	}

	// Test an empty manifest
	emptyManifest := Manifest{
		FileList: map[string]bool{
			"go.mod":            true, // all commented out
			"main.go":           true, // all commented out
			"internal/util.go":  true, // all commented out
			"cmd/cli.go":        true, // all commented out
			"README.md":         true, // all commented out
			"testdata/test.txt": true, // all commented out
		},
	}

	// Write empty manifest content to file
	if err := mockWriter.WriteManifest(emptyManifest, manifestFile); err != nil {
		t.Fatalf("Failed to write empty manifest file: %v", err)
	}

	// Process again with empty manifest
	isEmpty, err = processor.Process()
	if err != nil {
		t.Fatalf("Failed to process empty manifest: %v", err)
	}
	if !isEmpty {
		t.Errorf("Expected empty manifest result")
	}
}
