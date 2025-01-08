package utils

import (
	"log"
	"log/slog"
	"os"
)

var LogLevels = map[string]slog.Level{
	"DEBUG": slog.LevelDebug,
	"INFO":  slog.LevelInfo,
	"WARN":  slog.LevelWarn,
	"ERROR": slog.LevelError,
}

func IniciarLogger() {
	log.SetFlags(log.Ltime)

	file, err := os.Create("log.txt")
	if err != nil {
		panic(err)
	}
	log.SetOutput(file)
}
