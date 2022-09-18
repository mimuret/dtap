package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	stdLogger *zap.Logger
)

func New(levelStr string) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(levelStr)
	if err != nil {
		return nil, err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = level
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	zapLog, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	stdLogger = zapLog
	return stdLogger, nil
}

func GetLogger() *zap.Logger {
	return stdLogger
}

func init() {
	var err error
	stdLogger, err = New("info")
	if err != nil {
		panic("failed to create logger")
	}
}
