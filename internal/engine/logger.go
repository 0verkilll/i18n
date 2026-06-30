// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"context"
	"sync/atomic"

	"github.com/0verkilll/i18n/internal/core"
)

// Compile-time assertions that NopLogger implements Logger and LeveledLogger.
// NopLogger uses value receivers, so the assertion uses a value (not pointer).
var (
	_ core.Logger        = NopLogger{}
	_ core.LeveledLogger = NopLogger{}
)

// NopLogger is a silent logger that discards all messages.
// This is the default logger when no logger is configured.
// NopLogger is stateless and safe for concurrent use.
type NopLogger struct{}

// Debug discards the message.
func (NopLogger) Debug(string, ...any) {}

// Info discards the message.
func (NopLogger) Info(string, ...any) {}

// Warn discards the message.
func (NopLogger) Warn(string, ...any) {}

// Error discards the message.
func (NopLogger) Error(string, ...any) {}

// Fatal discards the message.
func (NopLogger) Fatal(string, ...any) {}

// WithFields returns the same NopLogger since fields are not used.
func (n NopLogger) WithFields(...any) core.Logger { return n }

// WithContext returns the same NopLogger since context is not used.
func (n NopLogger) WithContext(context.Context) core.Logger { return n }

// WithLevel returns the same NopLogger since level filtering is not used.
func (n NopLogger) WithLevel(core.LogLevel) core.Logger { return n }

// Enabled always returns false since NopLogger never logs.
func (NopLogger) Enabled(core.LogLevel) bool { return false }

// loggerBox wraps core.Logger so atomic.Value sees a consistent concrete
// type regardless of which implementation is stored. Standard Go idiom for
// atomic swap of an interface value.
type loggerBox struct{ l core.Logger }

// globalLogger is the package-level logger used by all Translator instances
// that don't have a custom logger configured. It defaults to NopLogger which
// discards all log messages, maintaining silent operation by default.
//
// The value is swapped atomically by SetLogger and read lock-free by GetLogger.
var globalLogger atomic.Value // holds loggerBox

func init() {
	globalLogger.Store(loggerBox{l: NopLogger{}})
}

// SetLogger sets the package-level logger.
// If l is nil, the logger is reset to NopLogger (silent operation).
// This logger will be used by all Translator instances that don't have
// a custom logger configured via WithLogger option.
//
// The Logger interface is compatible with github.com/0verkilll/logger.Logger,
// so you can pass any implementation from that package directly.
func SetLogger(l core.Logger) {
	if l == nil {
		globalLogger.Store(loggerBox{l: NopLogger{}})
		return
	}
	globalLogger.Store(loggerBox{l: l})
}

// GetLogger returns the package-level logger.
// By default, this returns NopLogger which discards all log messages.
func GetLogger() core.Logger {
	if v := globalLogger.Load(); v != nil {
		if b, ok := v.(loggerBox); ok && b.l != nil {
			return b.l
		}
	}
	return NopLogger{}
}
