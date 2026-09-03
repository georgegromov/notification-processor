package breaker

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

type Breaker struct {
	mu               sync.Mutex
	failures         int
	failureThreshold int
	openUntil        time.Time
	cooldown         time.Duration
	state            State
}

func New(failureThreshold int, cooldown time.Duration) *Breaker {
	return &Breaker{
		failureThreshold: failureThreshold,
		cooldown:         cooldown,
		state:            StateClosed,
	}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case StateOpen:
		if time.Now().After(b.openUntil) {
			b.state = StateHalfOpen
			return true
		}
		return false
	default:
		return true
	}
}

func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = StateClosed
}

func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.failureThreshold {
		b.state = StateOpen
		b.openUntil = time.Now().Add(b.cooldown)
	}
}
