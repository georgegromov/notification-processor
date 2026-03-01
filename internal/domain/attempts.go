package domain

// Attempts — счётчик попыток
type Attempts struct {
	// current - сколько попыток уже сделано
	current int
	// max - лимит попыток
	max int
}

// NewAttempts создаёт счётчик с заданным лимитом попыток (1..max включительно).
func NewAttempts(max int) Attempts {
	return Attempts{current: 0, max: max}
}

// Current возвращает количество уже выполненных попыток.
func (a Attempts) Current() int {
	return a.current
}

// CanRetry сообщает, можно ли делать ещё одну попытку.
func (a Attempts) CanRetry() bool {
	return a.current < a.max
}

// Increment возвращает счётчик после учёта очередной попытки.
func (a Attempts) Increment() Attempts {
	return Attempts{current: a.current + 1, max: a.max}
}

// Exhausted сообщает, что лимит попыток исчерпан.
func (a Attempts) Exhausted() bool {
	return a.current >= a.max
}
