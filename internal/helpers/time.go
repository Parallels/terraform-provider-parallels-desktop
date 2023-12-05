package helpers

import (
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	return time.ParseDuration(s)
}
