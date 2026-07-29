package logger

import "testing"

// The settings page can change the log level before InitLogger has run in a
// test or tooling context; that must not panic.
func TestSetLogLevelBeforeInit(t *testing.T) {
	for _, name := range []string{"DEBUG", "INFO", "WARN", "ERROR", "nonsense"} {
		SetLogLevel(name)
	}
}

func TestLogFunctionsBeforeInit(t *testing.T) {
	Info("info %d", 1)
	Warn("warn %s", "x")
	Error("error")
	Debug("debug")
}
