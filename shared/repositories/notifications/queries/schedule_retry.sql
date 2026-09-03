UPDATE notifications
SET status = 'pending',
    attempts = $2,
    next_retry_at = $3,
    locked_at = NULL,
    last_error = $4,
    updated_at = now()
WHERE id = $1 AND status = 'processing';