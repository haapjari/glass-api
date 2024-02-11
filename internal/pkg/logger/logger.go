package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel string

const (
	DebugLevel LogLevel = "DEBUG"
	InfoLevel  LogLevel = "INFO"
)

// Logger interface that abstracts logging functions.
type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

// NewLogger creates a new instance of a Logger with specified log level.
func NewLogger(level LogLevel) Logger {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "ts",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // ShortCallerEncoder includes the file and line number
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      true, // Enables DPanicLevel, and better stack traces
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Adjust log level based on the input
	switch level {
	case DebugLevel:
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case InfoLevel:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		panic("failed to initialize zap logger: " + err.Error())
	}

	return &zapLogger{logger: logger.Sugar()}
}

// zapLogger is an implementation of Logger interface using zap.
type zapLogger struct {
	logger *zap.SugaredLogger
}

// Debugf logs a debug message.
func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Errorf logs an error message.
func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// Infof logs an informational message.
func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warnf logs a warning message.
func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}
