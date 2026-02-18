package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/logger"
)

func TestNopLogger_ImplementsInterface(t *testing.T) {
	var l logger.Logger = logger.NopLogger{}
	assert.NotNil(t, l)
}

func TestNopLogger_Info(t *testing.T) {
	l := logger.NopLogger{}
	assert.NotPanics(t, func() {
		l.Info("info message")
	})
	assert.NotPanics(t, func() {
		l.Info("info message", "key", "value", 123)
	})
}

func TestNopLogger_Error(t *testing.T) {
	l := logger.NopLogger{}
	assert.NotPanics(t, func() {
		l.Error("error message")
	})
	assert.NotPanics(t, func() {
		l.Error("error message", "err", "something")
	})
}

func TestNopLogger_Debug(t *testing.T) {
	l := logger.NopLogger{}
	assert.NotPanics(t, func() {
		l.Debug("debug message")
	})
	assert.NotPanics(t, func() {
		l.Debug("debug message", "a", 1, "b", true)
	})
}

func TestNopLogger_ThroughInterface(t *testing.T) {
	var l logger.Logger = logger.NopLogger{}

	assert.NotPanics(t, func() {
		l.Info("info")
		l.Error("error")
		l.Debug("debug")
	})
}
