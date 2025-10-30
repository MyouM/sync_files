package logs

import (
	"fmt"
	"log/slog"
	"os"
)

func LogsInit() {
	logFile, err := os.Create("log.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	logger := slog.New(slog.NewTextHandler(logFile, nil))
	slog.SetDefault(logger)
}
