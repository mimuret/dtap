package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	zap.ReplaceGlobals(zapLog)
	return zapLog, nil
}
