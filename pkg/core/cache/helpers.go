package cache

import (
	"fmt"
	"log"
)

// SecToMin converts a duration in seconds to a formatted string (MM:SS or HH:MM:SS).
// It returns "0:00" for negative inputs and logs a warning.
func SecToMin(seconds int) string {
	if seconds < 0 {
		log.Println("Warning: SecToMin received a negative duration.")
		return "0:00"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}
