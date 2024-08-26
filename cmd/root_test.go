package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gkwa/nearwait/internal/logger"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCustomLogger(t *testing.T) {
	var buf bytes.Buffer

	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapLogger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zapConfig.EncoderConfig),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	))

	customLogger := zapr.NewLogger(zapLogger)

	cliLogger = customLogger

	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("Expected log output, but got none")
	}

	t.Logf("Log output: %s", logOutput)
}

func TestJSONLogger(t *testing.T) {
	oldVerbose, oldLogFormat := verbose, logFormat
	verbose, logFormat = true, "json"
	defer func() {
		verbose, logFormat = oldVerbose, oldLogFormat
	}()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	customLogger := logger.NewConsoleLogger(verbose, logFormat == "json")
	cliLogger = customLogger

	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy log output: %v", err)
	}
	logOutput := buf.String()

	if logOutput == "" {
		t.Error("Expected log output, but got none")
	}

	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	for _, line := range lines {
		var jsonMap map[string]interface{}
		err := json.Unmarshal([]byte(line), &jsonMap)
		if err != nil {
			t.Errorf("Expected valid JSON, but got error: %v", err)
		}
	}

	t.Logf("Log output: %s", logOutput)
}
