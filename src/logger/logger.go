package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	logger   *log.Logger
	logFile  *os.File
	logLevel int
	mutex    sync.Mutex
}

var (
	instance *Logger
	once     sync.Once
)

func GetInstance() *Logger {
	once.Do(func() {
		instance = &Logger{}
		instance.initLogger()
	})
	return instance
}

func (l *Logger) initLogger() {
	logDir := "logs"
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("Error creating logs directory: %v", err)
	}

	logFilePath := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}

	terminalWriter := &colorWriter{writer: os.Stdout}
	fileWriter := &colorWriter{writer: file, noColor: true}

	multiWriter := io.MultiWriter(terminalWriter, fileWriter)
	l.logger = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
	l.logFile = file
	l.logLevel = LevelInfo
}

func (l *Logger) log(level int, format string, v ...interface{}) {
	if level < l.logLevel {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()

	colorStart := GetColorStr(level)
	levelStr := GetLevelStr(level)
	colorEnd := "\033[0m"

	msg := fmt.Sprintf(format, v...)
	logMsg := fmt.Sprintf("%s %s", levelStr, msg)

	l.logger.Output(3, fmt.Sprintf("%s%s%s\n", colorStart, logMsg, colorEnd))
}

func GetLevelStr(level int) string {
	switch level {
	case LevelDebug:
		return "[DEBUG]"
	case LevelInfo:
		return "[INFO]"
	case LevelWarn:
		return "[WARN]"
	case LevelError:
		return "[ERROR]"
	default:
		return "[INFO]"
	}
}

func GetColorStr(level int) string {
	switch level {
	case LevelDebug:
		return "\033[32m"
	case LevelInfo:
		return "\033[34m"
	case LevelWarn:
		return "\033[33m"
	case LevelError:
		return "\033[31m"
	default:
		return "\033[37m"
	}
}

func Debug(format string, v ...interface{}) {
	GetInstance().log(LevelDebug, format, v...)
}

func Info(format string, v ...interface{}) {
	GetInstance().log(LevelInfo, format, v...)
}

func Warn(format string, v ...interface{}) {
	GetInstance().log(LevelWarn, format, v...)
}

func Error(format string, v ...interface{}) {
	GetInstance().log(LevelError, format, v...)
}

func SetLogLevel(level int) {
	GetInstance().logLevel = level
}

func ParseLogLevel(levelStr string) int {
	if strings.EqualFold("DEBUG", levelStr) {
		return LevelDebug
	}
	if strings.EqualFold("INFO", levelStr) {
		return LevelInfo
	}
	if strings.EqualFold("WARN", levelStr) {
		return LevelWarn
	}
	if strings.EqualFold("ERROR", levelStr) {
		return LevelError
	}
	return LevelInfo
}

func Close() {
	if instance != nil && instance.logFile != nil {
		instance.logFile.Close()
	}
}

type colorWriter struct {
	writer  io.Writer
	noColor bool
}

func (cw *colorWriter) Write(p []byte) (n int, err error) {
	if cw.noColor {
		p = stripColors(p)
	}
	return cw.writer.Write(p)
}

func stripColors(p []byte) []byte {
	var result []byte
	inColorCode := false
	for _, b := range p {
		if b == '\033' {
			inColorCode = true
		} else if b == 'm' && inColorCode {
			inColorCode = false
		} else if !inColorCode {
			result = append(result, b)
		}
	}
	return result
}
