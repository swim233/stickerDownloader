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
	// Initialized here rather than in InitLogger: a zero AtomicLevel holds a
	// nil pointer, so SetLogLevel would panic if it ran first.
	level = zap.NewAtomicLevelAt(zap.InfoLevel)
)

// InitLogger initializes the zap logger with console + file output.
func InitLogger() {
	level.SetLevel(zap.DebugLevel)

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

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
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

// fallback keeps logging from panicking before InitLogger has run — a log
// call must never be the thing that takes a request down.
func fallback(level, format string, v ...any) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", level, fmt.Sprintf(format, v...))
}

func Info(format string, v ...any) {
	if sugar == nil {
		fallback("INFO", format, v...)
		return
	}
	sugar.Infof(format, v...)
}

func Warn(format string, v ...any) {
	if sugar == nil {
		fallback("WARN", format, v...)
		return
	}
	sugar.Warnf(format, v...)
}

func Error(format string, v ...any) {
	if sugar == nil {
		fallback("ERROR", format, v...)
		return
	}
	sugar.Errorf(format, v...)
}

func Debug(format string, v ...any) {
	if sugar == nil {
		return
	}
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

func (a *BotAPILoggerAdapter) Printf(format string, v ...any) {
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

func (a *BotAPILoggerAdapter) Println(v ...any) {
	msg := fmt.Sprint(v...)
	a.Printf("%s", msg)
}
