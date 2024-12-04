package core

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
)

func TestGenerateWithNilFS(t *testing.T) {
	logger := zapr.NewLogger(zaptest.NewLogger(t))
	mg := NewManifestGenerator(logger)

	_, err := mg.Generate(false, ".nearwait.yml")
	if err == nil {
		t.Error("Generate() with nil FS should return error")
	}
}

func TestGenerateWithFS(t *testing.T) {
	logger := zapr.NewLogger(zaptest.NewLogger(t))
	mg := NewManifestGenerator(logger)

	fsys := fstest.MapFS{
		"test.txt": {Data: []byte("test")},
	}
	mg.WithFS(fsys)

	_, err := mg.Generate(false, ".nearwait.yml")
	if err != nil {
		t.Errorf("Generate() with valid FS failed: %v", err)
	}
}

func TestGenerateWithRealFS(t *testing.T) {
	logger := zapr.NewLogger(zaptest.NewLogger(t))
	mg := NewManifestGenerator(logger)
	mg.WithFS(os.DirFS("."))

	_, err := mg.Generate(false, ".nearwait.yml")
	if err != nil {
		t.Errorf("Generate() with real FS failed: %v", err)
	}
}
