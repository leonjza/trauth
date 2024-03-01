package trauth

import (
	"fmt"
	"io"
	"log"
	"os"
)

func NewLogger() *log.Logger {
	logger := log.New(io.Discard, "[trauth] ", log.Ldate|log.Ltime)
	logger.SetOutput(os.Stdout)

	return logger
}

// debugLog is a helper to print logs for debugging purposes
func debugLog(m string) {
	os.Stdout.WriteString(fmt.Sprintf(" --> [trauth debug] %s", m))
}
