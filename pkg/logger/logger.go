package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// L 全局日志实例
	L *zap.Logger
	// S 全局Sugar日志实例
	S *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level      string
	Format     string // json or console
	OutputPath string
}

// Init 初始化日志
func Init(cfg *Config) error {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath == "" || cfg.OutputPath == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)
	L = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	S = L.Sugar()

	return nil
}

// Sync 刷新日志缓冲
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}

// Debug 输出Debug级别日志
func Debug(msg string, fields ...zap.Field) {
	L.Debug(msg, fields...)
}

// Info 输出Info级别日志
func Info(msg string, fields ...zap.Field) {
	L.Info(msg, fields...)
}

// Warn 输出Warn级别日志
func Warn(msg string, fields ...zap.Field) {
	L.Warn(msg, fields...)
}

// Error 输出Error级别日志
func Error(msg string, fields ...zap.Field) {
	L.Error(msg, fields...)
}

// Fatal 输出Fatal级别日志并退出
func Fatal(msg string, fields ...zap.Field) {
	L.Fatal(msg, fields...)
}

// With 创建带有字段的子日志器
func With(fields ...zap.Field) *zap.Logger {
	return L.With(fields...)
}
