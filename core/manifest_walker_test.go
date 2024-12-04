package core

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetCurrentFiles(t *testing.T) {
	testCases := []struct {
		name          string
		files         []string
		excludeDirs   []string
		expectedFiles []string
		expectError   bool
	}{
		{
			name: "Basic file structure",
			files: []string{
				"file1.txt",
				"dir1/file2.txt",
				"dir2/file3.txt",
			},
			excludeDirs: []string{},
			expectedFiles: []string{
				"file1.txt",
				"dir1/file2.txt",
				"dir2/file3.txt",
			},
		},
		{
			name: "With excluded directory",
			files: []string{
				"file1.txt",
				"dir1/file2.txt",
				"excluded/file3.txt",
			},
			excludeDirs: []string{"excluded"},
			expectedFiles: []string{
				"file1.txt",
				"dir1/file2.txt",
			},
		},
		{
			name: "With nested excluded directory",
			files: []string{
				"file1.txt",
				"dir1/file2.txt",
				"dir1/excluded/file3.txt",
				"dir2/file4.txt",
			},
			excludeDirs: []string{"excluded"},
			expectedFiles: []string{
				"file1.txt",
				"dir1/file2.txt",
				"dir2/file4.txt",
			},
		},
		{
			name: "With multiple excluded directories",
			files: []string{
				"file1.txt",
				"exclude1/file2.txt",
				"dir1/exclude2/file3.txt",
				"dir2/file4.txt",
			},
			excludeDirs: []string{"exclude1", "exclude2"},
			expectedFiles: []string{
				"file1.txt",
				"dir2/file4.txt",
			},
		},
		{
			name: "With excluded directory as substring",
			files: []string{
				"file1.txt",
				"exclude/file2.txt",
				"not_excluded/file3.txt",
				"excluded/file4.txt",
				"dir1/exclude_nested/file5.txt",
				"dir2/excluded_nested/file6.txt",
			},
			excludeDirs: []string{"exclude"},
			expectedFiles: []string{
				"file1.txt",
				"not_excluded/file3.txt",
				"excluded/file4.txt",
				"dir1/exclude_nested/file5.txt",
				"dir2/excluded_nested/file6.txt",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := createTestFS(tc.files)

			config := zap.NewDevelopmentConfig()
			config.EncoderConfig.TimeKey = ""
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			zapLogger, _ := config.Build()
			logger := zapr.NewLogger(zapLogger)

			mg := NewManifestGenerator(logger)
			mg.excludeDirs = make(map[string]bool)
			for _, dir := range tc.excludeDirs {
				mg.excludeDirs[dir] = true
			}

			files, err := mg.getFilesFromFS(fsys)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				gotFiles := make([]string, 0, len(files))
				for file := range files {
					gotFiles = append(gotFiles, filepath.ToSlash(file))
				}

				sort.Strings(gotFiles)
				sort.Strings(tc.expectedFiles)

				if !reflect.DeepEqual(gotFiles, tc.expectedFiles) {
					t.Errorf("Files mismatch.\nExpected:\n%s\nGot:\n%s", formatFiles(tc.expectedFiles), formatFiles(gotFiles))
				}
			}
		})
	}
}

func createTestFS(files []string) fs.FS {
	fsys := fstest.MapFS{}
	for _, file := range files {
		fsys[file] = &fstest.MapFile{Data: []byte("test")}
	}
	return fsys
}

func formatFiles(files []string) string {
	return "  " + strings.Join(files, "\n  ")
}
