package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l, err := cfg.Build()
	if err != nil {
		return err
	}

	Logger = l

	return nil
}
