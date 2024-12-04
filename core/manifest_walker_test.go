package core

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
)

func testLogger(t *testing.T) logr.Logger {
	return zapr.NewLogger(zaptest.NewLogger(t))
}

func TestGetCurrentFiles(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]*fstest.MapFile
		exclude map[string]bool
		want    []string
	}{
		{
			name: "Basic file structure",
			files: map[string]*fstest.MapFile{
				"file1.txt":      {},
				"dir1/file2.txt": {},
				"dir2/file3.txt": {},
			},
			want: []string{"dir1/file2.txt", "dir2/file3.txt", "file1.txt"},
		},
		{
			name: "With excluded directory",
			files: map[string]*fstest.MapFile{
				"file1.txt":          {},
				"dir1/file2.txt":     {},
				"excluded/file3.txt": {},
			},
			exclude: map[string]bool{"excluded": true},
			want:    []string{"dir1/file2.txt", "file1.txt"},
		},
		{
			name: "With nested excluded directory",
			files: map[string]*fstest.MapFile{
				"file1.txt":               {},
				"dir1/file2.txt":          {},
				"dir1/excluded/file3.txt": {},
				"dir2/file4.txt":          {},
			},
			exclude: map[string]bool{"excluded": true},
			want:    []string{"dir1/file2.txt", "dir2/file4.txt", "file1.txt"},
		},
		{
			name: "With multiple excluded directories",
			files: map[string]*fstest.MapFile{
				"file1.txt":               {},
				"exclude1/file2.txt":      {},
				"dir1/exclude2/file3.txt": {},
				"dir2/file4.txt":          {},
			},
			exclude: map[string]bool{"exclude1": true, "exclude2": true},
			want:    []string{"dir2/file4.txt", "file1.txt"},
		},
		{
			name: "With excluded directory as substring",
			files: map[string]*fstest.MapFile{
				"file1.txt":                      {},
				"exclude/file2.txt":              {},
				"excluded/file4.txt":             {},
				"not_excluded/file3.txt":         {},
				"dir1/exclude_nested/file5.txt":  {},
				"dir2/excluded_nested/file6.txt": {},
			},
			exclude: map[string]bool{"exclude": true},
			want: []string{
				"dir1/exclude_nested/file5.txt",
				"dir2/excluded_nested/file6.txt",
				"excluded/file4.txt",
				"file1.txt",
				"not_excluded/file3.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS(tt.files)
			mg := NewManifestGenerator(testLogger(t))
			mg.excludeDirs = tt.exclude
			mg.WithFS(fsys)

			got, err := mg.GetCurrentFiles()
			if err != nil {
				t.Errorf("GetCurrentFiles() error = %v", err)
				return
			}

			var gotFiles []string
			for file := range got {
				gotFiles = append(gotFiles, filepath.ToSlash(file))
			}
			sort.Strings(gotFiles)

			want := make([]string, len(tt.want))
			copy(want, tt.want)
			sort.Strings(want)

			if len(gotFiles) != len(want) {
				t.Errorf("Files mismatch.\nExpected:\n%v\nGot:\n%v",
					joinNewlines(want),
					joinNewlines(gotFiles))
				return
			}

			for i := range want {
				if gotFiles[i] != want[i] {
					t.Errorf("Files mismatch.\nExpected:\n%v\nGot:\n%v",
						joinNewlines(want),
						joinNewlines(gotFiles))
					return
				}
			}
		})
	}
}

func joinNewlines(strs []string) string {
	var result string
	for _, s := range strs {
		result += s + "\n"
	}
	return result
}

// Mock error file system for testing
type errorFS struct{ err error }

func (e errorFS) Open(name string) (fs.File, error)          { return nil, e.err }
func (e errorFS) ReadDir(name string) ([]fs.DirEntry, error) { return nil, e.err }
func (e errorFS) Stat(name string) (fs.FileInfo, error)      { return nil, e.err }

func TestGetCurrentFilesError(t *testing.T) {
	mg := NewManifestGenerator(testLogger(t))
	mg.WithFS(errorFS{errors.New("mock error")})

	_, err := mg.GetCurrentFiles()
	if err == nil {
		t.Error("Expected error from GetCurrentFiles with error FS")
	}
}
