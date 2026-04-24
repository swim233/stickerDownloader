package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	level  zap.AtomicLevel
)

// InitLogger initializes the zap logger with console + file output.
func InitLogger() {
	level = zap.NewAtomicLevelAt(zap.DebugLevel)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "Time",
		LevelKey:         "Level",
		NameKey:          "Logger",
		CallerKey:        "Caller",
		MessageKey:       "Message",
		StacktraceKey:    "StackTrace",
		LineEnding:       zapcore.DefaultLineEnding,
		FunctionKey:      zapcore.OmitKey,
		ConsoleSeparator: "  ",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("[Zap] " + t.Format("2006/01/02 - 15:04:05"))
		},
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// File writer with rotation
	fileWriter := &lumberjack.Logger{
		Filename:   "./log/output.log",
		MaxSize:    20,
		MaxBackups: 5,
		Compress:   true,
		LocalTime:  true,
	}

	core := zapcore.NewCore(
		encoder,
		zap.CombineWriteSyncers(
			zapcore.AddSync(os.Stdout),
			zapcore.Lock(zapcore.AddSync(fileWriter)),
		),
		level,
	)

	logger = zap.New(core, zap.AddCaller())
	sugar = logger.Sugar()
}

// SetLogLevel changes the logger's log level dynamically.
func SetLogLevel(levelStr string) {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level.SetLevel(zap.DebugLevel)
	case "INFO":
		level.SetLevel(zap.InfoLevel)
	case "WARN":
		level.SetLevel(zap.WarnLevel)
	case "ERROR":
		level.SetLevel(zap.ErrorLevel)
	default:
		level.SetLevel(zap.InfoLevel)
	}
}

func Info(format string, v ...interface{}) {
	sugar.Infof(format, v...)
}

func Warn(format string, v ...interface{}) {
	sugar.Warnf(format, v...)
}

func Error(format string, v ...interface{}) {
	sugar.Errorf(format, v...)
}

func Debug(format string, v ...interface{}) {
	sugar.Debugf(format, v...)
}

// BotAPILoggerAdapter adapts zap for the telegram-bot-api logger interface.
type BotAPILoggerAdapter struct {
	logLevel zapcore.Level
}

func NewBotAPILoggerAdapter(levelStr string) *BotAPILoggerAdapter {
	var l zapcore.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		l = zap.DebugLevel
	case "WARN":
		l = zap.WarnLevel
	case "ERROR":
		l = zap.ErrorLevel
	default:
		l = zap.InfoLevel
	}
	return &BotAPILoggerAdapter{logLevel: l}
}

func (a *BotAPILoggerAdapter) Printf(format string, v ...interface{}) {
	if sugar == nil {
		return
	}
	switch a.logLevel {
	case zap.DebugLevel:
		sugar.Debugf(format, v...)
	case zap.WarnLevel:
		sugar.Warnf(format, v...)
	case zap.ErrorLevel:
		sugar.Errorf(format, v...)
	default:
		sugar.Infof(format, v...)
	}
}

func (a *BotAPILoggerAdapter) Println(v ...interface{}) {
	msg := fmt.Sprint(v...)
	a.Printf("%s", msg)
}
