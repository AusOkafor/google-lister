package logger

import (
	"log"
	"os"
)

type Logger struct {
	level string
}

func New(level string) *Logger {
	return &Logger{
		level: level,
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level == "debug" || l.level == "info" {
		log.Printf("[INFO] "+msg, args...)
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level == "debug" {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	log.Printf("[FATAL] "+msg, args...)
	os.Exit(1)
}
