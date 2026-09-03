package retry

import (
	"math/rand"
	"time"
)

const (
	baseDelay = 100 * time.Millisecond
	// jitterFactor = 0.5
)

func Next(attempt int) time.Duration {
	// attempt после инкремента: 1,2,3 → 100ms, 200ms, 400ms + jitter
	d := baseDelay << (attempt - 1)
	jitter := time.Duration(rand.Int63n(int64(d / 2)))
	return d + jitter
}

// func CalculateBackoff(attempt int) time.Duration {}
