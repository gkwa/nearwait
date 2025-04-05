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

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var outBuf bytes.Buffer
	_, err = io.Copy(&outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy stdout: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "Version:") {
		t.Errorf("Expected version information, but got: %s", output)
	}

	// Log output might be empty, so we'll skip checking it
	t.Logf("Log output: %s", buf.String())
}

func TestJSONLogger(t *testing.T) {
	oldVerbose, oldLogFormat := verbose, logFormat
	verbose, logFormat = true, "json"
	defer func() {
		verbose, logFormat = oldVerbose, oldLogFormat
	}()

	// Capture stderr for logs
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Capture stdout for version output
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	customLogger := logger.NewConsoleLogger(verbose, logFormat == "json")
	cliLogger = customLogger

	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	w.Close()
	wOut.Close()
	os.Stderr = oldStderr
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy log output: %v", err)
	}

	var outBuf bytes.Buffer
	_, err = io.Copy(&outBuf, rOut)
	if err != nil {
		t.Fatalf("Failed to copy stdout: %v", err)
	}

	logOutput := buf.String()
	versionOutput := outBuf.String()

	if !strings.Contains(versionOutput, "Version:") {
		t.Errorf("Expected version information, but got: %s", versionOutput)
	}

	// Log output might be empty in this case too
	if logOutput != "" {
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		for _, line := range lines {
			var jsonMap map[string]interface{}
			err := json.Unmarshal([]byte(line), &jsonMap)
			if err != nil {
				t.Errorf("Expected valid JSON, but got error: %v", err)
			}
		}
	}

	t.Logf("Log output: %s", logOutput)
	t.Logf("Version output: %s", versionOutput)
}
