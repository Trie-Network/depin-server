package utils

import (
	"fmt"
	"log"
	"time"
)

// LogInfo logs messages with UTC timestamp
func LogInfo(format string, args ...any) {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s\n", timestamp, message)
}
