package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/SisyphusSQ/summary-sys/utils/time_util"
)

type OutputMode string

const (
	OutputStdio  OutputMode = "stdio"
	OutputStderr OutputMode = "stderr"
	OutputFile   OutputMode = "file"
)

var Logger *ZapLogger

func New(isDebug bool, mode OutputMode, filePath ...string) error {
	logLevel := zapcore.InfoLevel
	if isDebug {
		logLevel = zapcore.DebugLevel
	}

	writeSyncer, err := buildWriteSyncer(mode, filePath...)
	if err != nil {
		return err
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    customLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(time_util.CSTLayout),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   customCallerEncoder,
	}

	encoder := zapcore.NewConsoleEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, writeSyncer, logLevel)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	Logger = NewZapLogger(logger)

	return nil
}

func buildWriteSyncer(mode OutputMode, filePath ...string) (zapcore.WriteSyncer, error) {
	switch mode {
	case OutputStdio:
		return zapcore.AddSync(os.Stdout), nil
	case OutputStderr:
		return zapcore.AddSync(os.Stderr), nil
	case OutputFile:
		logPath := defaultLogPath(filePath...)
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}

		rotating := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}
		return zapcore.AddSync(rotating), nil
	default:
		return nil, fmt.Errorf("unsupported log output mode: %s", mode)
	}
}

func defaultLogPath(filePath ...string) string {
	if len(filePath) == 0 {
		return "./logs/app.log"
	}

	candidate := strings.TrimSpace(filePath[0])
	if candidate == "" {
		return "./logs/app.log"
	}

	return candidate
}

func customLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if caller.Defined {
		enc.AppendString("[" + caller.TrimmedPath() + "]")
		return
	}
	enc.AppendString("[undefined]")
}

type ZapLogger struct {
	logger *zap.SugaredLogger
}

func NewZapLogger(logger *zap.SugaredLogger) *ZapLogger {
	return &ZapLogger{logger: logger}
}

func (l *ZapLogger) Debugf(format string, args ...any) {
	l.logger.Debugf(format, args...)
}

func (l *ZapLogger) Infof(format string, args ...any) {
	l.logger.Infof(format, args...)
}

func (l *ZapLogger) Warnf(format string, args ...any) {
	l.logger.Warnf(format, args...)
}

func (l *ZapLogger) Errorf(format string, args ...any) {
	l.logger.Errorf(format, args...)
}

func (l *ZapLogger) Sync() {
	_ = l.logger.Sync()
}
