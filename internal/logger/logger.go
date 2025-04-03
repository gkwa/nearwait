package logger

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	gcrLog "github.com/google/go-containerregistry/pkg/logs"
	"github.com/rs/zerolog"
	runtimeLog "sigs.k8s.io/controller-runtime/pkg/log"
)

func NewConsoleLogger(verbose, jsonFormat bool) logr.Logger {
	var zlog zerolog.Logger

	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		relPath, err := filepath.Rel(".", file)
		if err != nil {
			relPath = file
		}
		return relPath + ":" + strconv.Itoa(line)
	}

	if jsonFormat {
		zlog = zerolog.New(os.Stderr).With().Timestamp().Caller().Logger()
	} else {
		color.NoColor = !verbose
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    !verbose,
			TimeFormat: time.Kitchen,
		}

		if !verbose {
			consoleWriter.PartsExclude = []string{zerolog.TimestampFieldName}
		}

		zlog = zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()
	}

	if verbose {
		zlog = zlog.Level(zerolog.DebugLevel)
	} else {
		zlog = zlog.Level(zerolog.InfoLevel)
	}

	gcrLog.Warn.SetOutput(io.Discard)

	zerologr.VerbosityFieldName = "v"
	log := zerologr.New(&zlog)

	runtimeLog.SetLogger(log)

	return log
}
