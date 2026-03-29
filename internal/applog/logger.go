package applog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New() (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.Development = false
	cfg.DisableStacktrace = true
	cfg.DisableCaller = true
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	return cfg.Build()
}
