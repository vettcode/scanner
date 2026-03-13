package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit_DebugLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := Init("debug", buf)

	logger.Debug("debug message")
	assert.Contains(t, buf.String(), "debug message")
}

func TestInit_InfoLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := Init("info", buf)

	logger.Debug("should not appear")
	assert.Empty(t, buf.String())

	logger.Info("info message")
	assert.Contains(t, buf.String(), "info message")
}

func TestInit_WarnLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := Init("warn", buf)

	logger.Info("should not appear")
	assert.Empty(t, buf.String())

	logger.Warn("warn message")
	assert.Contains(t, buf.String(), "warn message")
}

func TestInit_ErrorLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := Init("error", buf)

	logger.Warn("should not appear")
	assert.Empty(t, buf.String())

	logger.Error("error message")
	assert.Contains(t, buf.String(), "error message")
}

func TestInit_DefaultLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := Init("invalid", buf)

	logger.Info("info message")
	assert.Contains(t, buf.String(), "info message")
}

func TestParseLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, ParseLevel("debug"))
	assert.Equal(t, slog.LevelInfo, ParseLevel("info"))
	assert.Equal(t, slog.LevelWarn, ParseLevel("warn"))
	assert.Equal(t, slog.LevelError, ParseLevel("error"))
	assert.Equal(t, slog.LevelInfo, ParseLevel("unknown"))
}
