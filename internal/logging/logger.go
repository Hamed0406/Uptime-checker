package logging

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger(logDir string) (*zap.Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(logDir, "uptime.log"),
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     14, // days
		Compress:   true,
	})
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "ts"
	core := zapcore.NewCore(zapcore.NewJSONEncoder(cfg), w, zap.InfoLevel)
	return zap.New(core), nil
}
