package trauth

import (
	"io"
	"log"
	"os"
)

func NewLogger() *log.Logger {
	logger := log.New(io.Discard, "trauth ", log.LstdFlags)
	logger.SetOutput(os.Stdout)

	return logger
}
